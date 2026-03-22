package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"goagent/internal/config"
	"goagent/internal/llm"
	"goagent/internal/tools/resources"
)

// DialogAgent represents an agent with tools.
type DialogAgent struct {
	id           string
	name         string
	description  string
	tools        *resources.AgentTools
	llmClient    *llm.Client
	systemPrompt string
}

// NewDialogAgent creates a new dialog agent with tools.
func NewDialogAgent(
	id, name, description string,
	toolConfig *resources.AgentToolConfig,
	llmClient *llm.Client,
	systemPrompt string,
) (*DialogAgent, error) {
	// Create agent tools with the given configuration
	agentTools := resources.NewAgentTools(toolConfig)

	// Log loaded tools for debugging
	agentTools.LogTools(name)

	return &DialogAgent{
		id:           id,
		name:         name,
		description:  description,
		tools:        agentTools,
		llmClient:    llmClient,
		systemPrompt: systemPrompt,
	}, nil
}

// Start initializes the agent.
func (da *DialogAgent) Start(ctx context.Context) error {
	slog.Info("Agent started",
		"id", da.id,
		"name", da.name,
		"tool_count", len(da.tools.ListTools()),
	)
	return nil
}

// Process handles user input and returns response.
func (da *DialogAgent) Process(ctx context.Context, userMessage string) (string, error) {
	// Check if user is asking about capabilities
	if da.isCapabilityQuestion(userMessage) {
		return da.listCapabilities(), nil
	}

	// Generate tool prompt with usage instructions
	toolPrompt := da.generateToolPrompt()

	// Build full prompt with system message and tool info
	fullPrompt := fmt.Sprintf("%s\n\n%s\n\nUser: %s\nAssistant:",
		da.systemPrompt,
		toolPrompt,
		userMessage,
	)

	// Call LLM with tool support (multi-round)
	var currentPrompt = fullPrompt
	var conversationHistory []string
	maxToolRounds := 3

	for round := 0; round < maxToolRounds; round++ {
		response, err := da.llmClient.Generate(ctx, currentPrompt)
		if err != nil {
			return "", fmt.Errorf("LLM generation failed: %w", err)
		}

		// Check if response contains tool calls
		if !strings.Contains(response, "[TOOL:") {
			return response, nil
		}

		// Execute tool calls
		toolResponse, err := da.handleToolCalls(ctx, response)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Append to conversation history
		conversationHistory = append(conversationHistory, fmt.Sprintf("Assistant: %s", response))
		conversationHistory = append(conversationHistory, fmt.Sprintf("Tool Result: %s", toolResponse))

		// Build next prompt with conversation history (without toolPrompt to reduce confusion)
		currentPrompt = fmt.Sprintf("%s\n\nUser: %s\n\n%s\n\nAssistant:",
			da.systemPrompt,
			userMessage,
			strings.Join(conversationHistory, "\n"),
		)
	}

	// If we've exceeded max rounds, get final response
	response, err := da.llmClient.Generate(ctx, currentPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	return response, nil
}

// isCapabilityQuestion checks if the user is asking about the agent's capabilities.
func (da *DialogAgent) isCapabilityQuestion(message string) bool {
	lowerMsg := strings.ToLower(message)

	questions := []string{
		"你会什么",
		"你会啥",
		"你都会啥",
		"你能",
		"你能做",
		"what can you do",
		"capabilities",
		"工具",
		"available tools",
		"what tools",
	}

	for _, q := range questions {
		if strings.Contains(lowerMsg, q) {
			return true
		}
	}

	return false
}

// generateToolPrompt generates a prompt string describing available tools with usage instructions.
func (da *DialogAgent) generateToolPrompt() string {
	schemas := da.tools.GetSchemas()

	if len(schemas) == 0 {
		return "No tools available."
	}

	var sb strings.Builder
	sb.WriteString("You have access to the following tools:\n\n")

	for _, schema := range schemas {
		fmt.Fprintf(&sb, "- %s (%s): %s\n", schema.Name, schema.Category, schema.Description)
		if len(schema.Parameters.GetProperties()) > 0 {
			sb.WriteString("  Parameters:\n")
			for paramName, param := range schema.Parameters.GetProperties() {
				paramInfo := fmt.Sprintf("    - %s (%s): %s", paramName, param.Type, param.Description)
				if len(param.Enum) > 0 {
					paramInfo += fmt.Sprintf(" (allowed values: %v)", param.Enum)
				}
				sb.WriteString(paramInfo + "\n")
			}
		}
	}

	sb.WriteString("\nIMPORTANT: When you need to use a tool to answer a question, you MUST use this exact format:\n")
	sb.WriteString("[TOOL:tool_name {\"param1\": \"value1\", \"param2\": 123}]\n\n")
	sb.WriteString("Examples:\n")
	sb.WriteString("- Get current time: [TOOL:datetime {\"operation\": \"now\"}]\n")
	sb.WriteString("- Add numbers: [TOOL:calculator {\"operation\": \"add\", \"operands\": [1, 2, 3]}]\n")
	sb.WriteString("- Multiply numbers: [TOOL:calculator {\"operation\": \"multiply\", \"operands\": [5, 6]}]\n")
	sb.WriteString("- Divide numbers: [TOOL:calculator {\"operation\": \"divide\", \"operands\": [10, 2]}]\n\n")
	sb.WriteString("Tool Usage Guide:\n")
	sb.WriteString("- calculator: Use mathematical formulas for large computations (e.g., sum 1..n → n*(n+1)/2). Avoid iterative calculations.\n")
	sb.WriteString("- datetime: Use for time operations. Always specify the 'operation' parameter.\n")
	sb.WriteString("- file_tools: Use absolute paths for file operations.\n")
	sb.WriteString("- json_tools: Use for JSON parsing and manipulation.\n\n")
	sb.WriteString("REMEMBER: Always check the required parameters for each tool and provide them correctly.")

	return sb.String()
}

// listCapabilities returns a formatted list of agent capabilities.
func (da *DialogAgent) listCapabilities() string {
	schemas := da.tools.GetSchemas()

	var sb strings.Builder
	fmt.Fprintf(&sb, "我是 %s (%s)\n\n", da.name, da.description)
	sb.WriteString("我有以下工具可以使用：\n\n")

	for _, schema := range schemas {
		fmt.Fprintf(&sb, "• %s (%s)\n", schema.Name, schema.Category)
		fmt.Fprintf(&sb, "  描述: %s\n", schema.Description)

		if len(schema.Parameters.GetProperties()) > 0 {
			sb.WriteString("  参数:\n")
			for paramName, param := range schema.Parameters.GetProperties() {
				fmt.Fprintf(&sb, "    - %s (%s): %s\n",
					paramName,
					param.Type,
					param.Description,
				)
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// handleToolCalls parses and executes tool calls from LLM response.
func (da *DialogAgent) handleToolCalls(ctx context.Context, response string) (string, error) {
	// Parse tool calls from response
	// Format: [TOOL:tool_name {"param": "value"}]
	lines := strings.Split(response, "\n")
	var results []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[TOOL:") {
			continue
		}

		// Extract tool name and params
		end := strings.Index(line, "]")
		if end == -1 {
			continue
		}

		content := strings.TrimPrefix(line[:end], "[TOOL:")
		parts := strings.SplitN(content, " ", 2)
		if len(parts) < 2 {
			continue
		}

		toolName := parts[0]
		paramsJSON := parts[1]

		// Execute tool (logging is handled by AgentTools.Execute)
		result, err := da.tools.Execute(ctx, toolName, paramsJSONToMap(paramsJSON))
		if err != nil {
			results = append(results, fmt.Sprintf("工具 %s 执行失败: %v", toolName, err))
			continue
		}

		// Get user-friendly formatted result from result metadata if available
		if len(result.Metadata) > 0 {
			if formatted, exists := result.Metadata["formatted"]; exists {
				results = append(results, formatted.(string))
			} else {
				results = append(results, fmt.Sprintf("工具 %s 执行完成", toolName))
			}
		} else {
			results = append(results, fmt.Sprintf("工具 %s 执行完成", toolName))
		}
	}

	return strings.Join(results, "\n\n"), nil
}

// paramsJSONToMap converts JSON string to map.
func paramsJSONToMap(jsonStr string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}

// ConversationSession manages a multi-agent conversation session.
type ConversationSession struct {
	agents        map[string]*DialogAgent
	agentSelector func(string) string
	memory        []string
	maxMemory     int
}

// NewConversationSession creates a new conversation session.
func NewConversationSession(agents []*DialogAgent) *ConversationSession {
	agentMap := make(map[string]*DialogAgent)
	for _, agent := range agents {
		agentMap[agent.id] = agent
	}

	return &ConversationSession{
		agents:    agentMap,
		memory:    make([]string, 0),
		maxMemory: 20,
	}
}

// SetAgentSelector sets a function to select which agent should respond.
func (cs *ConversationSession) SetAgentSelector(selector func(string) string) {
	cs.agentSelector = selector
}

// Process processes user input and returns agent response.
func (cs *ConversationSession) Process(ctx context.Context, userInput string) (string, error) {
	// Add user input to memory
	cs.memory = append(cs.memory, fmt.Sprintf("User: %s", userInput))
	if len(cs.memory) > cs.maxMemory {
		cs.memory = cs.memory[1:]
	}

	// Select agent
	var agentID string
	if cs.agentSelector != nil {
		agentID = cs.agentSelector(userInput)
	} else {
		agentID = cs.selectAgentByKeyword(userInput)
	}

	agent, exists := cs.agents[agentID]
	if !exists {
		return "", fmt.Errorf("agent %s not found", agentID)
	}

	// Process with agent
	response, err := agent.Process(ctx, userInput)
	if err != nil {
		return "", fmt.Errorf("agent processing failed: %w", err)
	}

	// Add agent response to memory
	cs.memory = append(cs.memory, fmt.Sprintf("%s (%s): %s", agent.name, agent.id, response))
	if len(cs.memory) > cs.maxMemory {
		cs.memory = cs.memory[1:]
	}

	return fmt.Sprintf("[%s]: %s", agent.name, response), nil
}

// selectAgentByKeyword selects agent based on user input keywords.
func (cs *ConversationSession) selectAgentByKeyword(input string) string {
	lowerInput := strings.ToLower(input)

	// Keywords for research agent
	researchKeywords := []string{"搜索", "查找", "分析", "信息", "新闻", "天气", "知识", "search", "find", "analyze", "information"}
	for _, keyword := range researchKeywords {
		if strings.Contains(lowerInput, keyword) {
			return "research-agent-1"
		}
	}

	// Default to tools agent
	return "tools-agent-1"
}

// ShowMemory returns the conversation memory.
func (cs *ConversationSession) ShowMemory() string {
	if len(cs.memory) == 0 {
		return "No conversation history."
	}

	var sb strings.Builder
	sb.WriteString("=== Conversation History ===\n\n")
	for _, msg := range cs.memory {
		sb.WriteString(msg + "\n")
	}
	return sb.String()
}

// createAgentsFromConfig creates agents based on configuration file.
func createAgentsFromConfig(cfg *config.Config, llmClient *llm.Client) ([]*DialogAgent, error) {
	var agents []*DialogAgent

	// Check if tools configuration is provided
	if len(cfg.Tools.Agents) == 0 {
		return nil, fmt.Errorf("no agent tool configuration found in config file")
	}

	// Create agents from configuration
	for agentID, agentConfig := range cfg.Tools.Agents {
		// Build system prompt with tool instructions
		systemPrompt := agentConfig.SystemPrompt
		if systemPrompt == "" {
			// Use default system prompt if not provided
			systemPrompt = fmt.Sprintf("You are a %s with access to various tools.\n"+
				"IMPORTANT: When the user asks about your capabilities, provide a detailed list of your tools.\n"+
				"When you need to use a tool, use this format: [TOOL:tool_name {\"param\": \"value\"}]",
				agentConfig.Name)
		}

		toolConfig := &resources.AgentToolConfig{
			Enabled: agentConfig.Tools,
		}

		agent, err := NewDialogAgent(
			agentID,
			agentConfig.Name,
			agentConfig.Description,
			toolConfig,
			llmClient,
			systemPrompt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent %s: %w", agentID, err)
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

func main() {
	slog.Info("Starting Multi-Agent Dialog Example")

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/server.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if err := config.LoadFromEnv(cfg); err != nil {
		slog.Error("Failed to load env config", "error", err)
		os.Exit(1)
	}

	// Create LLM client
	llmConfig := &llm.Config{
		Provider: cfg.LLM.Provider,
		APIKey:   cfg.LLM.APIKey,
		BaseURL:  cfg.LLM.BaseURL,
		Model:    cfg.LLM.Model,
		Timeout:  cfg.LLM.Timeout,
	}

	llmClient, err := llm.NewClient(llmConfig)
	if err != nil {
		slog.Error("Failed to create LLM client", "error", err)
		os.Exit(1)
	}

	// Register builtin tools once for all agents
	if err := resources.RegisterBuiltinToolsForAgent(); err != nil {
		slog.Error("Failed to register builtin tools", "error", err)
		os.Exit(1)
	}

	// Create agents from configuration file
	agents, err := createAgentsFromConfig(cfg, llmClient)
	if err != nil {
		slog.Error("Failed to create agents from config", "error", err)
		os.Exit(1)
	}

	if len(agents) == 0 {
		slog.Error("No agents created from configuration")
		os.Exit(1)
	}

	// Create conversation session
	session := NewConversationSession(agents)

	// Start agents
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, agent := range agents {
		if err := agent.Start(ctx); err != nil {
			slog.Error("Failed to start agent", "id", agent.id, "error", err)
		}
	}

	// Interactive conversation loop
	fmt.Println("=== Multi-Agent Dialog System ===")
	fmt.Println("Available agents:")
	for i, agent := range agents {
		fmt.Printf("  %d. %s (%s)\n", i+1, agent.name, agent.id)
	}
	fmt.Println("\nCommands:")
	fmt.Println("  'exit' - Exit the program")
	fmt.Println("  'memory' - Show conversation history")
	fmt.Println("  'agents' - List all agents and their capabilities")
	fmt.Println("  '@agent-id' - Direct message to specific agent")
	fmt.Println("")
	fmt.Println("Starting conversation... (type 'exit' to quit)")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "memory":
			fmt.Println(session.ShowMemory())
			continue
		case "agents":
			for _, agent := range agents {
				fmt.Printf("\n--- %s Capabilities ---\n", agent.name)
				fmt.Println(agent.listCapabilities())
			}
			continue
		}

		// Check for direct agent messaging
		if strings.HasPrefix(input, "@") {
			parts := strings.SplitN(input[1:], " ", 2)
			if len(parts) > 1 {
				agentID := parts[0]
				message := parts[1]
				agent, exists := session.agents[agentID]
				if exists {
					response, err := agent.Process(ctx, message)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
					fmt.Printf("[%s]: %s\n", agent.name, response)
					continue
				}
			}
		}

		// Process through session
		startTime := time.Now()
		response, err := session.Process(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("%s\n", response)
		fmt.Printf("(Response time: %v)\n\n", elapsed.Round(time.Millisecond))
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Scanner error", "error", err)
	}
}
