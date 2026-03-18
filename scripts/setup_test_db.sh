#!/bin/bash

# Setup test database for repository tests

echo "Connecting to test database..."
go run -mod=mod <<'EOF'
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"goagent/internal/storage/postgres"
)

func main() {
	ctx := context.Background()

	// Connect to database
	connStr := "host=localhost port=5433 user=postgres password=postgres dbname=styleagent sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully")

	// Enable pgvector extension
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		log.Fatalf("Failed to create vector extension: %v", err)
	}
	fmt.Println("Enabled pgvector extension")

	// Create pool
	pool := postgres.NewPool(db)

	// Run migrations
	if err := postgres.MigrateStorage(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Migrations completed successfully")
}
EOF