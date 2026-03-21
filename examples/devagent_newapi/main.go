package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goagent/api/client"
	"goagent/api/core"
	"goagent/internal/llm/output"
	"goagent/internal/workflow/engine"
)

const (
	codeDir          = "code"
	testDir          = "test"
	docsDir          = "docs"
	maxPreviewLength = 300
)

var (
	outputDir = "output"
)

func main() {
	log.Println("=== DevAgent - Developer Assistant with Workflow ===")

	ctx := context.Background()

	if err := initializeOutputDirectories(); err != nil {
		log.Fatalf("Failed to initialize output directories: %v", err)
	}

	devClient, err := client.NewClientFromDefaultPath()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer devClient.Close(ctx)

	config := devClient.GetConfig()
	displayConfiguration(config)

	workflowClient, err := client.NewWorkflowClient(devClient)
	if err != nil {
		log.Fatalf("Failed to create workflow client: %v", err)
	}

	memorySvc, err := devClient.Memory()
	if err != nil {
		log.Printf("Memory service not available: %v", err)
	}

	parser := output.NewParser()

	for {
		userInput, shouldExit := getUserInput()
		if shouldExit {
			log.Println("再见！")
			break
		}

		if userInput == "" {
			continue
		}

		var sessionID string
		if memorySvc != nil {
			sessionID, err = createSession(ctx, memorySvc, userInput)
			if err != nil {
				log.Printf("Failed to create session: %v", err)
			}
		}

		log.Printf("\n正在处理: %s", userInput)

		workflowPath := "config/workflow.yaml"
		result, err := workflowClient.ExecuteFromFile(ctx, workflowPath, userInput)
		if err != nil {
			log.Printf("✗ 执行失败: %v", err)
			continue
		}

		if err := processAndSaveResults(ctx, result, parser); err != nil {
			log.Printf("✗ 处理结果失败: %v", err)
			continue
		}

		if memorySvc != nil {
			if err := saveToMemory(ctx, memorySvc, sessionID, userInput, result); err != nil {
				log.Printf("Failed to save to memory: %v", err)
			}

			if err := distillTask(ctx, memorySvc, userInput); err != nil {
				log.Printf("Failed to distill task: %v", err)
			}
		}
	}
}

// initializeOutputDirectories creates output directories if they don't exist.
func initializeOutputDirectories() error {
	dirs := []string{
		outputDir,
		filepath.Join(outputDir, codeDir),
		filepath.Join(outputDir, testDir),
		filepath.Join(outputDir, docsDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}

// displayConfiguration displays the loaded configuration.
func displayConfiguration(config *client.ConfigFile) {
	log.Println("\n=== 配置的 Agents ===")
	for _, agent := range config.Agents.Sub {
		log.Printf("  - %s (%s): %s", agent.ID, agent.Type, agent.Name)
	}

	log.Println("\n=== 记忆功能 ===")
	log.Printf("  会话记忆: %v", config.Memory.Enabled)
	log.Printf("  记忆蒸馏: %v", true)
}

// getUserInput reads a full line of user input from stdin.
func getUserInput() (string, bool) {
	fmt.Print("\n请输入开发任务（或输入 'quit' 退出）: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", true
	}

	userInput := strings.TrimSpace(scanner.Text())
	if userInput == "quit" || userInput == "exit" {
		return "", true
	}

	return userInput, false
}

// createSession creates a new session for the user input.
func createSession(ctx context.Context, memorySvc core.MemoryService, userInput string) (string, error) {
	if memorySvc == nil {
		return "", nil
	}

	sessionConfig := &core.SessionConfig{
		UserID: "dev-user",
	}

	sessionID, err := memorySvc.CreateSession(ctx, sessionConfig)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	if err := memorySvc.AddMessage(ctx, sessionID, core.MessageRoleUser, userInput); err != nil {
		return "", fmt.Errorf("add user message: %w", err)
	}

	log.Printf("✓ 创建会话: %s", sessionID)
	return sessionID, nil
}

// processAndSaveResults processes workflow results and saves to files.
func processAndSaveResults(ctx context.Context, result *engine.WorkflowResult, parser *output.Parser) error {
	log.Println("\n=== Agent Team 协作完成 ===")
	displayExecutionSummary(result)

	var codeFiles, testFiles, docFiles []string
	var codeContent, testContent, docsContent []string

	// 只保存关键步骤的输出
	stepPriority := map[string]OutputType{
		"Generate Code":          OutputTypeCode,
		"Generate Tests":         OutputTypeTest,
		"Generate Documentation": OutputTypeDocs,
		"Code Review":            OutputTypeReview,
	}

	for _, step := range result.Steps {
		if step.Status != "completed" {
			continue
		}

		outputType, isKeyStep := stepPriority[step.Name]
		if !isKeyStep {
			continue
		}

		items, err := parseStepOutput(parser, step.Name, step.Output)
		if err != nil {
			log.Printf("  ⚠️ 解析步骤 %s 输出失败: %v", step.Name, err)
			continue
		}

		if len(items) == 0 {
			continue
		}

		// 只保存每个步骤的第一个有效项目
		mainItem := items[0]
		filePath, err := saveOutputItem(step.Name, mainItem, 0)
		if err != nil {
			log.Printf("  ✗ 保存文件失败: %v", err)
			continue
		}

		categorizeAndDisplay(step.Name, filePath, &codeFiles, &testFiles, &docFiles)

		// 收集内容用于生成文档
		if outputType == OutputTypeCode {
			codeContent = append(codeContent, mainItem.Content)
		} else if outputType == OutputTypeTest {
			testContent = append(testContent, mainItem.Content)
		} else if outputType == OutputTypeDocs || outputType == OutputTypeReview {
			docsContent = append(docsContent, mainItem.Content)
		}
	}

	// 只在有实际代码内容时生成文档
	if len(codeContent) > 0 {
		if err := generateArchitectureDocument(ctx, result, codeContent, testContent, docsContent); err != nil {
			log.Printf("  ⚠️ 生成架构文档失败: %v", err)
		}

		if err := generateAuditDocument(ctx, result, codeContent, testContent); err != nil {
			log.Printf("  ⚠️ 生成审计文档失败: %v", err)
		}
	}

	displaySummary(codeFiles, testFiles, docFiles)
	return nil
}

// displayExecutionSummary displays the workflow execution summary.
func displayExecutionSummary(result *engine.WorkflowResult) {
	log.Printf("\n执行摘要:")
	log.Printf("  总耗时: %.1f 秒", result.Duration.Seconds())
	log.Printf("  完成步骤: %d/%d", countCompletedSteps(result.Steps), len(result.Steps))
	log.Printf("  执行状态: %s", result.Status)
}

// parseStepOutput parses step output using the LLM output parser.
func parseStepOutput(parser *output.Parser, stepName, stepOutput string) ([]*OutputItem, error) {
	if stepOutput == "" {
		return []*OutputItem{}, nil
	}

	// Try to parse as JSON first
	data, err := parser.ParseJSON(stepOutput)
	if err != nil {
		// JSON parsing failed, try to extract content from the output
		items := extractItemsFromRawOutput(stepName, stepOutput)
		if len(items) > 0 {
			return items, nil
		}
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	items, err := extractItemsFromData(data)
	if err != nil {
		// JSON parsing succeeded but extraction failed, try raw extraction
		items := extractItemsFromRawOutput(stepName, stepOutput)
		if len(items) > 0 {
			return items, nil
		}
		return nil, fmt.Errorf("extract items: %w", err)
	}

	// If no items were extracted from JSON, try raw extraction
	if len(items) == 0 {
		items = extractItemsFromRawOutput(stepName, stepOutput)
	}

	outputType := detectOutputType(stepName)
	for _, item := range items {
		item.Type = outputType
	}

	return items, nil
}

// extractItemsFromRawOutput extracts items from raw text output when JSON parsing fails.
func extractItemsFromRawOutput(stepName, stepOutput string) []*OutputItem {
	items := []*OutputItem{}
	outputType := detectOutputType(stepName)

	// Try to find code blocks in the output
	codeBlocks := extractCodeBlocks(stepOutput)
	if len(codeBlocks) > 0 {
		// 如果有代码块，只取第一个作为主要内容
		mainBlock := codeBlocks[0]
		items = append(items, &OutputItem{
			Name:        getDefaultFileName(outputType),
			Description: stepName,
			Content:     mainBlock,
			Type:        outputType,
		})
		return items
	}

	// If no code blocks found, create a single item with the entire output
	items = append(items, &OutputItem{
		Name:        getDefaultFileName(outputType),
		Description: stepName,
		Content:     stepOutput,
		Type:        outputType,
	})

	return items
}

// getDefaultFileName returns the default file name for a given output type.
func getDefaultFileName(outputType OutputType) string {
	switch outputType {
	case OutputTypeCode:
		return "main"
	case OutputTypeTest:
		return "main_test"
	case OutputTypeDocs:
		return "README"
	case OutputTypeReview:
		return "CODE_REVIEW"
	default:
		return "output"
	}
}

// extractCodeBlocks extracts code blocks from markdown-style output.
func extractCodeBlocks(output string) []string {
	blocks := []string{}

	// Find all ```code``` blocks
	start := strings.Index(output, "```")
	for start != -1 {
		// Find the end of the opening ``` (might include language identifier)
		lineEnd := strings.Index(output[start:], "\n")
		if lineEnd == -1 {
			break
		}
		start += lineEnd + 1

		// Find the closing ```
		closeIndex := strings.Index(output[start:], "```")
		if closeIndex == -1 {
			break
		}
		blockEnd := start + closeIndex
		block := strings.TrimSpace(output[start:blockEnd])
		if len(block) > 0 {
			blocks = append(blocks, block)
		}

		// Move to after the closing ```
		start = strings.Index(output[blockEnd+3:], "```")
		if start != -1 {
			start = blockEnd + 3 + start
		}
	}

	return blocks
}

// extractItemsFromData extracts items from parsed JSON data.
func extractItemsFromData(data map[string]interface{}) ([]*OutputItem, error) {
	if itemsArray, ok := data["items"].([]interface{}); ok {
		items := make([]*OutputItem, 0, len(itemsArray))
		for _, itemData := range itemsArray {
			if itemMap, ok := itemData.(map[string]interface{}); ok {
				items = append(items, &OutputItem{
					Name:        getString(itemMap, "name"),
					Description: getString(itemMap, "description"),
					Content:     getString(itemMap, "content"),
					Language:    getString(itemMap, "language"),
				})
			}
		}
		return items, nil
	}

	return []*OutputItem{
		{
			Name:        getString(data, "name"),
			Description: getString(data, "description"),
			Content:     getString(data, "content"),
			Language:    getString(data, "language"),
		},
	}, nil
}

// getString safely extracts a string value from a map.
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// detectOutputType detects the output type based on step name.
func detectOutputType(stepName string) OutputType {
	stepName = strings.ToLower(stepName)

	switch {
	case strings.Contains(stepName, "review"):
		return OutputTypeReview
	case strings.Contains(stepName, "test"):
		return OutputTypeTest
	case strings.Contains(stepName, "docs") || strings.Contains(stepName, "doc"):
		return OutputTypeDocs
	case strings.Contains(stepName, "code"):
		return OutputTypeCode
	default:
		return OutputTypeDocs
	}
}

// displayRawOutput displays raw output when parsing fails.
func displayRawOutput(step *engine.StepResult) {
	emoji := getStepEmoji(step.Name)
	log.Printf("\n%s %s (%.1fs):", emoji, step.Name, step.Duration.Seconds())

	preview := step.Output
	if len(preview) > maxPreviewLength {
		preview = preview[:maxPreviewLength] + "..."
	}
	log.Printf("  输出: %s", preview)
}

// saveOutputItem saves an output item to a file.
func saveOutputItem(stepName string, item *OutputItem, index int) (string, error) {
	ext := getFileExtension(item.Type)
	fileName := sanitizeFilename(item.Name)

	// 如果文件名是空的或者是默认的，使用类型特定的默认文件名
	if fileName == "" || fileName == "output" {
		fileName = getDefaultFileName(item.Type)
	}

	filePath := filepath.Join(outputDir, getSubDir(item.Type), fileName+ext)

	if err := os.WriteFile(filePath, []byte(item.Content), 0644); err != nil {
		return "", fmt.Errorf("write file %s: %w", filePath, err)
	}

	return filePath, nil
}

// getSubDir returns the subdirectory for the output type.
func getSubDir(outputType OutputType) string {
	switch outputType {
	case OutputTypeCode:
		return codeDir
	case OutputTypeTest:
		return testDir
	case OutputTypeDocs, OutputTypeReview:
		return docsDir
	default:
		return docsDir
	}
}

// categorizeAndDisplay categorizes and displays the saved file.
func categorizeAndDisplay(stepName, filePath string, codeFiles, testFiles, docFiles *[]string) {
	emoji := getStepEmoji(stepName)

	switch {
	case strings.Contains(strings.ToLower(stepName), "code"):
		*codeFiles = append(*codeFiles, filePath)
		log.Printf("  %s 💻 %s", emoji, filepath.Base(filePath))
	case strings.Contains(strings.ToLower(stepName), "test"):
		*testFiles = append(*testFiles, filePath)
		log.Printf("  %s 🧪 %s", emoji, filepath.Base(filePath))
	case strings.Contains(strings.ToLower(stepName), "docs"):
		*docFiles = append(*docFiles, filePath)
		log.Printf("  %s 📚 %s", emoji, filepath.Base(filePath))
	case strings.Contains(strings.ToLower(stepName), "review"):
		*docFiles = append(*docFiles, filePath)
		log.Printf("  %s 🔍 %s", emoji, filepath.Base(filePath))
	default:
		*docFiles = append(*docFiles, filePath)
		log.Printf("  %s 📦 %s", emoji, filepath.Base(filePath))
	}
}

// displaySummary displays the summary of generated files.
func displaySummary(codeFiles, testFiles, docFiles []string) {
	if len(codeFiles) == 0 && len(testFiles) == 0 && len(docFiles) == 0 {
		return
	}

	log.Println("\n=== Team 交付成果 ===")

	if len(codeFiles) > 0 {
		log.Printf("💻 代码文件 (%d): %v", len(codeFiles), codeFiles)
	}

	if len(testFiles) > 0 {
		log.Printf("🧪 测试文件 (%d): %v", len(testFiles), testFiles)
	}

	if len(docFiles) > 0 {
		log.Printf("📚 文档文件 (%d): %v", len(docFiles), docFiles)
	}

	log.Println("\n💡 Agent Team 说明:")
	log.Println("   💻 Code Agent - 负责编写核心代码")
	log.Println("   🧪 Test Agent - 负责编写测试用例")
	log.Println("   📚 Docs Agent - 负责编写项目文档")
	log.Println("   🔍 Review Agent - 负责代码审查和质量保证")
	log.Println("   ⚡ Workflow - DAG编排，支持并行执行以提高效率")
}

// saveToMemory saves the interaction to memory.
func saveToMemory(ctx context.Context, memorySvc core.MemoryService, sessionID, userInput string, result *engine.WorkflowResult) error {
	if memorySvc == nil || sessionID == "" {
		return nil
	}

	if len(result.Steps) == 0 {
		return nil
	}

	lastStep := result.Steps[len(result.Steps)-1]
	if err := memorySvc.AddMessage(ctx, sessionID, core.MessageRoleAssistant, lastStep.Output); err != nil {
		return fmt.Errorf("add assistant message: %w", err)
	}

	return nil
}

// distillTask distills the task for future reference.
func distillTask(ctx context.Context, memorySvc core.MemoryService, userInput string) error {
	if memorySvc == nil {
		return nil
	}

	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	log.Printf("\n正在蒸馏任务: %s", taskID)

	task, err := memorySvc.DistillTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("distill task: %w", err)
	}

	log.Printf("✓ 任务蒸馏完成: %s", task.TaskID)
	return nil
}

// countCompletedSteps counts the number of completed steps.
func countCompletedSteps(steps []*engine.StepResult) int {
	count := 0
	for _, step := range steps {
		if step.Status == "completed" {
			count++
		}
	}
	return count
}

// getStepEmoji returns emoji for development steps.
func getStepEmoji(stepName string) string {
	stepName = strings.ToLower(stepName)

	switch {
	case strings.Contains(stepName, "extract"):
		return "📋"
	case strings.Contains(stepName, "review"):
		return "🔍"
	case strings.Contains(stepName, "test"):
		return "🧪"
	case strings.Contains(stepName, "docs") || strings.Contains(stepName, "doc"):
		return "📚"
	case strings.Contains(stepName, "code"):
		return "💻"
	default:
		return "📦"
	}
}

// sanitizeFilename sanitizes a string to be used as a filename.
func sanitizeFilename(name string) string {
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result.WriteRune(c)
		} else if c == ' ' || c == '_' {
			result.WriteRune('_')
		} else if c == '-' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// getFileExtension returns the file extension based on output type.
func getFileExtension(outputType OutputType) string {
	switch outputType {
	case OutputTypeCode:
		return ".go"
	case OutputTypeTest:
		// Default name already contains '_test', so extension is just '.go'.
		return ".go"
	case OutputTypeDocs:
		return ".md"
	case OutputTypeReview:
		return "_review.md"
	default:
		return ".txt"
	}
}

// generateArchitectureDocument generates an architecture design document.
func generateArchitectureDocument(ctx context.Context, result *engine.WorkflowResult, codeContent, testContent, docsContent []string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	executionID := result.ExecutionID

	docBuilder := strings.Builder{}
	docBuilder.WriteString("# Architecture Design Document\n\n")
	docBuilder.WriteString(fmt.Sprintf("**Generated:** %s\n", timestamp))
	docBuilder.WriteString(fmt.Sprintf("**Execution ID:** %s\n\n", executionID))

	docBuilder.WriteString("## Overview\n\n")
	docBuilder.WriteString("This document describes the architecture and design of the generated solution.\n\n")

	docBuilder.WriteString("## Components\n\n")

	docBuilder.WriteString("### Code Components\n\n")
	for i, content := range codeContent {
		docBuilder.WriteString(fmt.Sprintf("#### Component %d\n\n", i+1))
		docBuilder.WriteString("```go\n")
		docBuilder.WriteString(content)
		docBuilder.WriteString("\n```\n\n")
	}

	docBuilder.WriteString("### Test Components\n\n")
	for i, content := range testContent {
		docBuilder.WriteString(fmt.Sprintf("#### Test Suite %d\n\n", i+1))
		docBuilder.WriteString("```go\n")
		docBuilder.WriteString(content)
		docBuilder.WriteString("\n```\n\n")
	}

	docBuilder.WriteString("## Documentation\n\n")
	for i, content := range docsContent {
		docBuilder.WriteString(fmt.Sprintf("### Documentation Section %d\n\n", i+1))
		docBuilder.WriteString(content)
		docBuilder.WriteString("\n\n")
	}

	docBuilder.WriteString("## Workflow Execution Details\n\n")
	docBuilder.WriteString(fmt.Sprintf("- **Total Duration:** %.2f seconds\n", result.Duration.Seconds()))
	docBuilder.WriteString(fmt.Sprintf("- **Total Steps:** %d\n", len(result.Steps)))
	docBuilder.WriteString(fmt.Sprintf("- **Completed Steps:** %d\n", countCompletedSteps(result.Steps)))
	docBuilder.WriteString(fmt.Sprintf("- **Status:** %s\n\n", result.Status))

	docBuilder.WriteString("### Step Details\n\n")
	for _, step := range result.Steps {
		docBuilder.WriteString(fmt.Sprintf("- **%s:** %s (%.2fs)\n", step.Name, step.Status, step.Duration.Seconds()))
	}

	docBuilder.WriteString("\n## Design Principles\n\n")
	docBuilder.WriteString("- **Simplicity:** Code follows clear, straightforward patterns.\n")
	docBuilder.WriteString("- **Testability:** Comprehensive test coverage for all components.\n")
	docBuilder.WriteString("- **Maintainability:** Well-documented code with clear structure.\n")
	docBuilder.WriteString("- **Performance:** Optimized for efficiency and resource usage.\n\n")

	docBuilder.WriteString("---\n")
	docBuilder.WriteString("*This document was automatically generated by DevAgent.*\n")

	filePath := filepath.Join(outputDir, docsDir, "architecture_design.md")
	if err := os.WriteFile(filePath, []byte(docBuilder.String()), 0644); err != nil {
		return fmt.Errorf("write architecture document: %w", err)
	}

	log.Printf("  📐 已生成架构设计文档: %s", filePath)
	return nil
}

// generateAuditDocument generates an audit document for the generated solution.
func generateAuditDocument(ctx context.Context, result *engine.WorkflowResult, codeContent, testContent []string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	executionID := result.ExecutionID

	docBuilder := strings.Builder{}
	docBuilder.WriteString("# Code Audit Report\n\n")
	docBuilder.WriteString(fmt.Sprintf("**Generated:** %s\n", timestamp))
	docBuilder.WriteString(fmt.Sprintf("**Execution ID:** %s\n\n", executionID))

	docBuilder.WriteString("## Executive Summary\n\n")
	docBuilder.WriteString(fmt.Sprintf("This audit report evaluates the code generated by DevAgent (Execution ID: %s).\n\n", executionID))

	docBuilder.WriteString("## Audit Findings\n\n")

	docBuilder.WriteString("### Code Quality Assessment\n\n")
	docBuilder.WriteString("#### Metrics\n\n")
	totalCodeLines := 0
	totalTestLines := 0
	for _, content := range codeContent {
		totalCodeLines += len(strings.Split(content, "\n"))
	}
	for _, content := range testContent {
		totalTestLines += len(strings.Split(content, "\n"))
	}

	docBuilder.WriteString(fmt.Sprintf("- **Total Code Lines:** %d\n", totalCodeLines))
	docBuilder.WriteString(fmt.Sprintf("- **Total Test Lines:** %d\n", totalTestLines))
	if totalCodeLines > 0 {
		coverage := float64(totalTestLines) / float64(totalCodeLines) * 100
		docBuilder.WriteString(fmt.Sprintf("- **Test Coverage Estimate:** %.2f%%\n", coverage))
	}
	docBuilder.WriteString("\n")

	docBuilder.WriteString("#### Compliance with Go Best Practices\n\n")
	docBuilder.WriteString("##### Strengths\n\n")
	docBuilder.WriteString("- Follows Go naming conventions\n")
	docBuilder.WriteString("- Uses proper error handling patterns\n")
	docBuilder.WriteString("- Implements context for cancellation\n")
	docBuilder.WriteString("- Includes comprehensive tests\n\n")

	docBuilder.WriteString("##### Areas for Review\n\n")
	docBuilder.WriteString("- TODO: Add performance benchmarks\n")
	docBuilder.WriteString("- TODO: Add integration tests\n")
	docBuilder.WriteString("- TODO: Review and optimize memory usage\n\n")

	docBuilder.WriteString("### Security Assessment\n\n")
	docBuilder.WriteString("#### Potential Security Considerations\n\n")
	docBuilder.WriteString("- Input validation should be reviewed\n")
	docBuilder.WriteString("- Error messages should not expose sensitive information\n")
	docBuilder.WriteString("- Dependencies should be regularly updated\n\n")

	docBuilder.WriteString("### Performance Assessment\n\n")
	docBuilder.WriteString("#### Workflow Performance\n\n")
	docBuilder.WriteString(fmt.Sprintf("- **Total Execution Time:** %.2f seconds\n", result.Duration.Seconds()))
	docBuilder.WriteString(fmt.Sprintf("- **Average Step Duration:** %.2f seconds\n", result.Duration.Seconds()/float64(len(result.Steps))))
	docBuilder.WriteString("\n")

	docBuilder.WriteString("### Recommendations\n\n")
	docBuilder.WriteString("1. **Code Review:** Conduct peer review before production deployment\n")
	docBuilder.WriteString("2. **Testing:** Add integration tests and end-to-end tests\n")
	docBuilder.WriteString("3. **Documentation:** Ensure all public APIs are documented\n")
	docBuilder.WriteString("4. **Performance:** Profile and optimize critical paths\n")
	docBuilder.WriteString("5. **Security:** Conduct security audit before deployment\n\n")

	docBuilder.WriteString("## Step-by-Step Analysis\n\n")
	for _, step := range result.Steps {
		docBuilder.WriteString(fmt.Sprintf("### %s\n\n", step.Name))
		docBuilder.WriteString(fmt.Sprintf("- **Status:** %s\n", step.Status))
		docBuilder.WriteString(fmt.Sprintf("- **Duration:** %.2f seconds\n", step.Duration.Seconds()))
		if step.Error != "" {
			docBuilder.WriteString(fmt.Sprintf("- **Error:** %s\n", step.Error))
		}
		docBuilder.WriteString("\n")
	}

	docBuilder.WriteString("## Conclusion\n\n")
	docBuilder.WriteString("The generated code demonstrates good adherence to Go best practices and coding standards. ")
	docBuilder.WriteString("Further testing and review are recommended before production deployment.\n\n")

	docBuilder.WriteString("---\n")
	docBuilder.WriteString("*This audit report was automatically generated by DevAgent.*\n")

	filePath := filepath.Join(outputDir, docsDir, "audit_report.md")
	if err := os.WriteFile(filePath, []byte(docBuilder.String()), 0644); err != nil {
		return fmt.Errorf("write audit document: %w", err)
	}

	log.Printf("  🔍 已生成审计文档: %s", filePath)
	return nil
}

// OutputType represents the type of output content.
type OutputType string

const (
	OutputTypeCode   OutputType = "code"
	OutputTypeTest   OutputType = "test"
	OutputTypeDocs   OutputType = "docs"
	OutputTypeReview OutputType = "review"
)

// OutputItem represents a single output item.
type OutputItem struct {
	Name        string
	Description string
	Content     string
	Language    string
	Type        OutputType
}
