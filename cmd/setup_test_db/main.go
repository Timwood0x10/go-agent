// Package main provides database setup for testing.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"goagent/internal/storage/postgres"
)

func main() {
	ctx := context.Background()

	// Create database config
	cfg := &postgres.Config{
		Host:            "localhost",
		Port:            5433,
		User:            "postgres",
		Password:        "postgres",
		Database:        "goagent",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		QueryTimeout:    2 * time.Second,
	}

	// Create pool
	pool, err := postgres.NewPool(cfg)
	if err != nil {
		slog.Error("Failed to create pool", "error", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			slog.Error("Failed to close pool", "error", err)
		}
	}()

	fmt.Println("Connected to database successfully")

	// Enable pgvector extension
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		slog.Error("Failed to create vector extension", "error", err)
	}
	fmt.Println("Enabled pgvector extension")

	// Run migrations
	if err := postgres.MigrateStorage(ctx, pool); err != nil {
		slog.Error("Failed to run migrations", "error", err)
	}
	fmt.Println("Migrations completed successfully")
}
