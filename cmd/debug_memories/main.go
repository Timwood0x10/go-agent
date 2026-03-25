package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"goagent/internal/storage/postgres"
	"goagent/internal/storage/postgres/repositories"
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

	// Create distilled memory repository
	distilledRepo := repositories.NewDistilledMemoryRepository(pool.GetDB(), pool.GetDB())

	// Set tenant context
	setQuery := fmt.Sprintf("SET app.tenant_id TO '%s'", "default")
	if _, err := pool.Exec(ctx, setQuery); err != nil {
		slog.Error("Failed to set tenant context", "error", err)
	} else {
		slog.Info("Tenant context set to 'default'")
	}

	// Test 1: Get all memories for user "ken"
	fmt.Println("\n=== Test 1: GetByUserID for 'ken' ===")
	memories, err := distilledRepo.GetByUserID(ctx, "default", "ken", 10)
	if err != nil {
		slog.Error("Failed to get memories for 'ken'", "error", err)

	} else {
		fmt.Printf("✓ Found %d memories for user 'ken'\n", len(memories))
		for i, mem := range memories {
			fmt.Printf("\n  Memory %d:\n", i+1)
			fmt.Printf("    ID: %s\n", mem.ID)
			fmt.Printf("    User ID: %s\n", mem.UserID)
			fmt.Printf("    Type: %s\n", mem.MemoryType)
			fmt.Printf("    Importance: %.2f\n", mem.Importance)
			fmt.Printf("    Content: %s\n", mem.Content)
			fmt.Printf("    Expires At: %s\n", mem.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Test 2: Get all memories (using raw SQL to bypass RLS)
	fmt.Println("\n=== Test 2: Raw SQL query (bypass RLS) ===")
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, memory_type, importance, content, expires_at, created_at
		FROM distilled_memories
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		slog.Error("✗ Failed to query all memories", "error", err)
	} else {
		defer func() {
			if err := rows.Close(); err != nil {
				slog.Error("✗ Failed to close rows", "error", err)
			}
		}()
		fmt.Println("✓ All memories in database:")
		for rows.Next() {
			var id, userID, memType, content string
			var importance float64
			var expiresAt, createdAt time.Time
			if err := rows.Scan(&id, &userID, &memType, &importance, &content, &expiresAt, &createdAt); err != nil {
				log.Printf("  ✗ Failed to scan row: %v", err)
				continue
			}
			fmt.Printf("\n  - ID: %s\n", id)
			fmt.Printf("    User ID: %s\n", userID)
			fmt.Printf("    Type: %s\n", memType)
			fmt.Printf("    Importance: %.2f\n", importance)
			fmt.Printf("    Content: %s\n", content)
			fmt.Printf("    Created: %s\n", createdAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Test 3: Check RLS status
	fmt.Println("\n=== Test 3: RLS Status ===")
	var rlsEnabled bool
	err = pool.QueryRow(ctx, `
		SELECT relrowsecurity FROM pg_class WHERE relname = 'distilled_memories'
	`).Scan(&rlsEnabled)
	if err != nil {
		log.Printf("✗ Failed to check RLS status: %v", err)
	} else {
		fmt.Printf("✓ RLS Enabled: %v\n", rlsEnabled)
	}

	// Test 4: Check tenant setting
	fmt.Println("\n=== Test 4: Current tenant setting ===")
	var currentTenant string
	err = pool.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&currentTenant)
	if err != nil {
		log.Printf("✗ Failed to get tenant setting: %v", err)
	} else {
		if currentTenant == "" {
			fmt.Println("✓ Tenant not set (empty)")
		} else {
			fmt.Printf("✓ Current tenant: %s\n", currentTenant)
		}
	}
}
