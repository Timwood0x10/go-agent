// Package main provides database setup for testing.
package main

import (
	"context"
	"fmt"
	"log"
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
		Database:        "styleagent",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		QueryTimeout:    2 * time.Second,
	}

	// Create pool
	pool, err := postgres.NewPool(cfg)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			log.Fatal("Failed to close pool: ", err)
		}
	}()

	fmt.Println("Connected to database successfully")

	// Enable pgvector extension
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		log.Fatalf("Failed to create vector extension: %v", err)
	}
	fmt.Println("Enabled pgvector extension")

	// Run migrations
	if err := postgres.MigrateStorage(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Migrations completed successfully")
}
