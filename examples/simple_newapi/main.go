// Package main demonstrates usage of the new layered API for GoAgent.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"goagent/api/client"
	"goagent/api/core"
)

func main() {
	log.Println("=== GoAgent New Layered API Example ===")

	// Load configuration from file (or use default path)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/server.yaml"
	}

	// Create client directly from config file - simple and clean!
	goagentClient, err := client.NewClientFromConfigPath(configPath)
	if err != nil {
		log.Fatalf("Failed to create client from config %s: %v", configPath, err)
	}
	defer goagentClient.Close(context.Background())

	log.Printf("✓ Client initialized successfully from: %s", configPath)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer goagentClient.Close(context.Background())

	// Check services availability
	if goagentClient.Ping(context.Background()) {
		log.Println("✓ All services are available")
	}

	// Example 1: Agent Management
	fmt.Println("\n--- Example 1: Agent Management ---")
	exampleAgentManagement(context.Background(), goagentClient)

	// Example 2: Memory Management
	fmt.Println("\n--- Example 2: Memory Management ---")
	exampleMemoryManagement(context.Background(), goagentClient)

	// Example 3: LLM Operations
	fmt.Println("\n--- Example 3: LLM Operations ---")
	exampleLLMOperations(context.Background(), goagentClient)

	// Example 4: Knowledge Retrieval
	fmt.Println("\n--- Example 4: Knowledge Retrieval ---")
	exampleKnowledgeRetrieval(context.Background(), goagentClient)

	log.Println("\n=== Example completed successfully ===")
}

// exampleAgentManagement demonstrates agent creation and management.
func exampleAgentManagement(ctx context.Context, client *client.Client) {
	agentSvc, err := client.Agent()
	if err != nil {
		log.Printf("Agent service not configured: %v", err)
		return
	}

	// Create different types of agents
	agentTypes := []struct {
		id   string
		name string
		typ  string
	}{
		{"agent-top-1", "Top Wear Recommender", "agent_top"},
		{"agent-bottom-1", "Bottom Wear Recommender", "agent_bottom"},
		{"agent-shoes-1", "Shoes Recommender", "agent_shoes"},
	}

	for _, agentInfo := range agentTypes {
		agent, err := agentSvc.CreateAgent(ctx, &core.AgentConfig{
			ID:   agentInfo.id,
			Name: agentInfo.name,
			Type: agentInfo.typ,
			Config: map[string]interface{}{
				"category": agentInfo.name,
				"version":  "1.0.0",
			},
		})
		if err != nil {
			log.Printf("Failed to create agent %s: %v", agentInfo.id, err)
			continue
		}
		log.Printf("✓ Created agent: %s (%s)", agent.ID, agent.Name)
	}

	// List all agents
	agents, pagination, err := agentSvc.ListAgents(ctx, &core.AgentFilter{
		Pagination: &core.PaginationRequest{
			Page:     1,
			PageSize: 10,
		},
	})
	if err != nil {
		log.Printf("Failed to list agents: %v", err)
		return
	}

	log.Printf("✓ Total agents: %d (page %d/%d)", pagination.Total, pagination.Page, pagination.TotalPages)
	for _, agent := range agents {
		log.Printf("  - %s: %s [%s]", agent.ID, agent.Name, agent.Status)
	}
}

// exampleMemoryManagement demonstrates session and message management.
func exampleMemoryManagement(ctx context.Context, client *client.Client) {
	memorySvc, err := client.Memory()
	if err != nil {
		log.Printf("Memory service not configured: %v", err)
		return
	}

	// Create a session for a user
	sessionID, err := memorySvc.CreateSession(ctx, &core.SessionConfig{
		UserID:    "user-001",
		TenantID:  "tenant-001",
		ExpiresIn: 24 * time.Hour,
	})
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return
	}
	log.Printf("✓ Created session: %s", sessionID)

	// Simulate a conversation
	conversation := []struct {
		role    core.MessageRole
		content string
	}{
		{core.MessageRoleUser, "I'm looking for clothes suitable for daily commuting"},
		{core.MessageRoleAssistant, "Great, I'll help you with some commuter outfit recommendations"},
		{core.MessageRoleUser, "I prefer casual style, budget 500-1000 yuan"},
		{core.MessageRoleAssistant, "Understood, I'll recommend options based on your budget and style preference"},
	}

	for _, msg := range conversation {
		err := memorySvc.AddMessage(ctx, sessionID, msg.role, msg.content)
		if err != nil {
			log.Printf("Failed to add message: %v", err)
			continue
		}
		log.Printf("✓ Added [%s] message: %s", msg.role, msg.content[:min(30, len(msg.content))]+"...")
	}

	// Retrieve conversation history
	messages, err := memorySvc.GetMessages(ctx, sessionID, &core.PaginationRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
		return
	}

	log.Printf("✓ Retrieved %d messages from session", len(messages))
	for i, msg := range messages {
		log.Printf("  %d. [%s]: %s", i+1, msg.Role, msg.Content)
	}
}

// exampleLLMOperations demonstrates LLM text generation.
func exampleLLMOperations(ctx context.Context, client *client.Client) {
	llmSvc, err := client.LLM()
	if err != nil {
		log.Printf("LLM service not configured: %v", err)
		return
	}

	if !llmSvc.IsEnabled() {
		log.Println("LLM service is not available")
		return
	}

	log.Printf("✓ LLM Provider: %s", llmSvc.GetProvider())
	log.Printf("✓ LLM Model: %s", llmSvc.GetModel())

	// Generate recommendations using LLM
	prompt := `Based on the following user preferences, recommend clothing:

User preferences: Casual style, suitable for daily commute, budget 500-1000 yuan

Please recommend 3 tops, strictly return in the following JSON format:
{
  "items": [
    {"name": "Product name", "price": price, "reason": "Recommendation reason"}
  ]
}`

	response, err := llmSvc.GenerateSimple(ctx, prompt)
	if err != nil {
		log.Printf("Failed to generate text: %v", err)
		return
	}

	log.Printf("✓ Generated recommendations:\n%s", response)

	// Generate embedding for semantic search
	embeddingResp, err := llmSvc.GenerateEmbedding(ctx, &core.EmbeddingRequest{
		Input: "Casual style commuter outfit",
	})
	if err != nil {
		log.Printf("Failed to generate embedding: %v", err)
		return
	}

	log.Printf("✓ Generated embedding with %d dimensions", len(embeddingResp.Embedding))
}

// exampleKnowledgeRetrieval demonstrates knowledge base operations.
func exampleKnowledgeRetrieval(ctx context.Context, client *client.Client) {
	retrievalSvc, err := client.Retrieval()
	if err != nil {
		log.Printf("Retrieval service not configured: %v", err)
		return
	}

	// Add knowledge items to the knowledge base
	knowledgeItems := []*core.KnowledgeItem{
		{
			TenantID: "tenant-001",
			Content:  "Casual style is suitable for daily commuting, choose comfortable and breathable fabrics like cotton or linen",
			Source:   "style-guide",
			Category: "clothing-tips",
			Tags:     []string{"casual", "commute", "fabric"},
		},
		{
			TenantID: "tenant-001",
			Content:  "Budget 500-1000 yuan can choose basic styles from fast fashion or light luxury brands",
			Source:   "budget-guide",
			Category: "budget-tips",
			Tags:     []string{"budget", "shopping", "brand"},
		},
		{
			TenantID: "tenant-001",
			Content:  "Commuter outfits recommend neutral color schemes like black, white, and gray, which are easy to match",
			Source:   "color-guide",
			Category: "color-tips",
			Tags:     []string{"commute", "color", "matching"},
		},
	}

	for _, item := range knowledgeItems {
		created, err := retrievalSvc.AddKnowledge(ctx, item)
		if err != nil {
			log.Printf("Failed to add knowledge: %v", err)
			continue
		}
		log.Printf("✓ Added knowledge item: %s", created.ID)
	}

	// Search for relevant knowledge
	query := "casual commuter outfit suggestions"
	results, err := retrievalSvc.Search(ctx, "tenant-001", query)
	if err != nil {
		log.Printf("Failed to search knowledge: %v", err)
		return
	}

	log.Printf("✓ Found %d knowledge items for query: '%s'", len(results), query)
	for i, result := range results {
		log.Printf("  %d. [Score: %.2f] %s", i+1, result.Score, result.Content)
	}

	// List all knowledge
	_, pagination, err := retrievalSvc.ListKnowledge(ctx, "tenant-001", &core.KnowledgeFilter{
		Category: "clothing-tips",
	})
	if err != nil {
		log.Printf("Failed to list knowledge: %v", err)
		return
	}

	log.Printf("✓ Total knowledge items in 'clothing-tips': %d", pagination.Total)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
