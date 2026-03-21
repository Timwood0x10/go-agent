package main

import (
	"context"
	"log"

	"goagent/api/client"
)

func main() {
	log.Println("=== GoAgent Simple Demo ===")

	// Step 1: Create client (just one line!)
	// It will automatically load config from config.yaml
	client, err := client.NewSimpleClient("config.yaml")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(context.Background())

	log.Println("✓ Client created successfully")

	// Step 2: Execute a query (just one line!)
	// No need to understand agents, memory, retrieval, etc.
	result, err := client.Execute(context.Background(), "I want to buy some casual shirts for daily commute, budget 500-1000 yuan")
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	log.Println("\n=== Result ===")
	log.Println(result)

	// Step 3: Optional - Use specific agent
	agentResult, err := client.ExecuteWithAgent(context.Background(), "agent-top", "Recommend 3 casual shirts")
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
	} else {
		log.Println("\n=== Agent Result ===")
		log.Println(agentResult)
	}

	log.Println("\n=== Demo completed ===")
}
