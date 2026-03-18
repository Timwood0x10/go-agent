// Package repositories provides test helper functions for repository tests.
package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// getTestDB returns a test database connection.
// This function connects to local Docker PostgreSQL container.
func getTestDB(t *testing.T) *sql.DB {
	host := "localhost"
	port := "5433"
	user := "postgres"
	password := "postgres"
	dbname := "styleagent"

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	log.Printf("Connected to test database: %s", dbname)
	return db
}

// cleanupTestDB cleans up test data from database.
func cleanupTestDB(t *testing.T, db *sql.DB) {
	tables := []string{
		"knowledge_chunks_1024",
		"experiences_1024",
		"tools",
		"conversations",
		"task_results_1024",
		"secrets",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to clean up table %s: %v", table, err)
		}
	}
}

// closeTestDB closes the test database connection.
func closeTestDB(t *testing.T, db *sql.DB) {
	if err := db.Close(); err != nil {
		t.Logf("Warning: Failed to close test database: %v", err)
	}
}

// getEnv returns environment variable value or default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestRepositories creates and initializes test repositories.
// This function sets up all repositories for testing with a shared database connection.
func setupTestRepositories(t *testing.T) (*sql.DB, *KnowledgeRepository, *ExperienceRepository, *SecretRepository) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	t.Cleanup(func() {
		cleanupTestDB(t, db)
		closeTestDB(t, db)
	})

	// Create repositories
	knowledgeRepo := NewKnowledgeRepository(db, db)
	experienceRepo := NewExperienceRepository(db)
	secretRepo := NewSecretRepository(db, make([]byte, 32))

	return db, knowledgeRepo, experienceRepo, secretRepo
}
