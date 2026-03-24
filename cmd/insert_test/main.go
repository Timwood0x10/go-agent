package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"goagent/internal/storage/postgres"
)

func main() {
	// Create database configuration
	dbConfig := &postgres.Config{
		Host:            "127.0.0.1",
		Port:            5433,
		User:            "postgres",
		Password:        "postgres",
		Database:        "goagent",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		QueryTimeout:    30 * time.Second,
		Embedding:       postgres.DefaultEmbeddingConfig(),
	}

	// Create database connection pool
	pool, err := postgres.NewPool(dbConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			slog.Error("Failed to close database pool", "error", err)
		}
	}()

	ctx := context.Background()

	// Test: Insert a memory with explicit user_id
	fmt.Println("=== Test: Insert memory with user_id ===")

	// Set tenant context
	setQuery := fmt.Sprintf("SET app.tenant_id TO '%s'", "default")
	if _, err := pool.Exec(ctx, setQuery); err != nil {
		log.Printf("Failed to set tenant context: %v", err)
		return
	}
	fmt.Println("✓ Tenant context set to 'default'")

	// Create empty embedding vector (1024 dimensions)
	emptyEmbedding := make([]float64, 1024)
	embeddingStr := postgres.FormatVector(emptyEmbedding)

	// Insert test memory
	testID := uuid.New().String()
	testContent := "Test memory for Ken"
	testUserID := "ken"
	_, err = pool.Exec(ctx, `
		INSERT INTO distilled_memories
		(id, tenant_id, user_id, session_id, content, embedding, embedding_model,
		 embedding_version, memory_type, importance, metadata, access_count,
		 last_accessed_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '{}'::jsonb, $11, NULL, $12, $13)
	`, testID, "default", testUserID, "test_session", testContent, embeddingStr, "test_model", 1,
		"profile", 0.75, 0, time.Now().Add(90*24*time.Hour), time.Now())

	if err != nil {
		log.Printf("✗ Failed to insert test memory: %v", err)
		return
	}
	fmt.Printf("✓ Inserted test memory: ID=%s, user_id=%s\n", testID, testUserID)

	// Verify insertion
	var userID string
	err = pool.QueryRow(ctx, `
		SELECT user_id FROM distilled_memories WHERE id = $1
	`, testID).Scan(&userID)

	if err != nil {
		log.Printf("✗ Failed to query inserted memory: %v", err)
	} else {
		fmt.Printf("✓ Verified: user_id in database = '%s'\n", userID)
		if userID != testUserID {
			fmt.Printf("✗ ERROR: user_id mismatch! Expected '%s', got '%s'\n", testUserID, userID)
		}
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM distilled_memories WHERE id = $1", testID)
	fmt.Println("✓ Cleanup: deleted test memory")
}
