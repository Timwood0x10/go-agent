package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"goagent/api/client"
	"goagent/internal/workflow/engine"
)

func main() {
	log.Println("=== GoAgent Fashion Recommendation System with Workflow ===")

	// Step 1: Load config and create client
	goagentClient, err := client.NewClientFromDefaultPath()
	if err != nil {
		slog.Error("Failed to create client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := goagentClient.Close(context.Background()); err != nil {
			slog.Error("Failed to close GOAGENT CLient", "error", err)
		}
	}()

	config := goagentClient.GetConfig()

	// Show configured agents
	log.Println("\n=== Configured Agents ===")
	for _, agent := range config.Agents.Sub {
		log.Printf("  - %s (%s): %s", agent.ID, agent.Type, agent.Name)
	}

	// User query
	userQuery := "I want casual clothes for daily commute, budget 500-1000 yuan, prefer black and white"
	log.Printf("\n=== User Query ===\n%s\n", userQuery)

	// Step 2: Create workflow client
	workflowClient, err := client.NewWorkflowClient(goagentClient)
	if err != nil {
		slog.Error("Failed to create workflow client", "error", err)
		os.Exit(1)
	}

	// Step 3: Load and execute workflow
	log.Println("\n=== Executing Workflow ===")
	workflowPath := "config/workflow.yaml"

	result, err := workflowClient.ExecuteFromFile(context.Background(), workflowPath, userQuery)
	if err != nil {
		slog.Error("Failed to execute workflow", "error", err)
		os.Exit(1)
	}

	// Step 4: Display results
	log.Println("\n=== Workflow Execution Result ===")
	log.Printf("Execution ID: %s", result.ExecutionID)
	log.Printf("Status: %s", result.Status)
	log.Printf("Duration: %v", result.Duration)
	log.Printf("Total Steps: %d", len(result.Steps))

	// Show each step result
	log.Println("\n=== 推荐结果 ===")

	// Parse and display recommendations by category
	for _, step := range result.Steps {
		if step.Status != "completed" {
			continue
		}

		items := parseRecommendations(step.Output)
		if len(items) == 0 {
			continue
		}

		// Get emoji based on step name
		emoji := getStepEmoji(step.Name)

		log.Printf("\n%s %s:", emoji, step.Name)

		for i, item := range items {
			if i >= 3 { // Only show top 3
				break
			}

			priceStr := ""
			if item.Price > 0 {
				priceStr = fmt.Sprintf(" - ¥%.0f", item.Price)
			}

			log.Printf("  %d. %s%s", i+1, item.Name, priceStr)
			if item.Reason != "" {
				log.Printf("     理由：%s", item.Reason)
			}
		}
	}

	// Show summary
	log.Printf("\n✓ 为您生成了 %d 类推荐，耗时 %.1f 秒", countCompletedSteps(result.Steps), result.Duration.Seconds())

	log.Println("\n=== Done! ===")
}

// parseRecommendations parses JSON output and extracts recommendation items.
func parseRecommendations(output string) []RecommendationItem {
	// Try to extract JSON from output
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return []RecommendationItem{}
	}

	jsonStr := output[jsonStart : jsonEnd+1]

	var result struct {
		Items []struct {
			Name   string  `json:"name"`
			Price  float64 `json:"price"`
			Reason string  `json:"reason"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []RecommendationItem{}
	}

	items := make([]RecommendationItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, RecommendationItem{
			Name:   item.Name,
			Price:  item.Price,
			Reason: item.Reason,
		})
	}

	return items
}

// RecommendationItem represents a parsed recommendation item.
type RecommendationItem struct {
	Name   string
	Price  float64
	Reason string
}

// getStepEmoji returns an emoji based on step name.
func getStepEmoji(stepName string) string {
	stepName = strings.ToLower(stepName)

	switch {
	case strings.Contains(stepName, "top") || strings.Contains(stepName, "上衣"):
		return "👕"
	case strings.Contains(stepName, "bottom") || strings.Contains(stepName, "下装"):
		return "👖"
	case strings.Contains(stepName, "shoe") || strings.Contains(stepName, "鞋"):
		return "👞"
	case strings.Contains(stepName, "accessory") || strings.Contains(stepName, "配饰"):
		return "🎒"
	case strings.Contains(stepName, "extract") || strings.Contains(stepName, "提取"):
		return "📋"
	case strings.Contains(stepName, "aggregate") || strings.Contains(stepName, "聚合"):
		return "✨"
	default:
		return "📦"
	}
}

// countCompletedSteps counts how many steps completed successfully.
func countCompletedSteps(steps []*engine.StepResult) int {
	count := 0
	for _, step := range steps {
		if step.Status == "completed" {
			count++
		}
	}
	return count
}
