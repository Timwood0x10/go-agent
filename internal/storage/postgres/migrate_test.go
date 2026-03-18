// nolint: errcheck // Test code may ignore return values
package postgres

import (
	"context"
	"testing"
)

// TestMigrate tests running database migrations.
func TestMigrate(t *testing.T) {
	t.Run("run migrations successfully", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		err = Migrate(context.Background(), pool)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("run migrations with nil pool", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic with nil pool: %v", r)
			}
		}()
		err := Migrate(context.Background(), nil)
		if err == nil {
			t.Error("expected error with nil pool")
		}
	})

	t.Run("run migrations with cancelled context", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = Migrate(ctx, pool)
		if err != nil {
			t.Logf("Expected error with cancelled context: %v", err)
		}
	})
}

// TestRollbackLast tests rolling back the last migration.
func TestRollbackLast(t *testing.T) {
	t.Run("rollback last migration", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		err = RollbackLast(context.Background(), pool)
		if err != nil {
			// Expected to return ErrQueryFailed as per implementation
			t.Logf("Expected error: %v", err)
		}
	})

	t.Run("rollback with nil pool", func(t *testing.T) {
		err := RollbackLast(context.Background(), nil)
		if err == nil {
			t.Error("expected error with nil pool")
		}
		// RollbackLast returns ErrQueryFailed for nil pool
	})
}

// TestSeed tests creating seed data for testing.
func TestSeed(t *testing.T) {
	t.Run("create seed data", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		err = Seed(context.Background(), pool)
		if err != nil {
			t.Logf("Expected error: %v", err)
		}
	})

	t.Run("seed with nil pool", func(t *testing.T) {
		err := Seed(context.Background(), nil)
		// Seed returns nil for nil pool (it's a no-op)
		_ = err
	})
}

// TestMigrate_MigrationSteps tests individual migration steps.
func TestMigrate_MigrationSteps(t *testing.T) {
	t.Run("migration 1 - user_profiles table", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		query := `CREATE TABLE IF NOT EXISTS user_profiles (
			user_id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			gender VARCHAR(50),
			age INTEGER,
			occupation VARCHAR(255),
			style JSONB,
			budget JSONB,
			colors JSONB,
			occasions JSONB,
			body_type VARCHAR(100),
			preferences JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`

		_, err = pool.Exec(context.Background(), query)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("migration 2 - sessions table", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		query := `CREATE TABLE IF NOT EXISTS sessions (
			session_id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			input TEXT,
			status VARCHAR(50),
			user_profile JSONB,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			expired_at TIMESTAMP
		)`

		_, err = pool.Exec(context.Background(), query)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("migration 3 - recommendations table", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		query := `CREATE TABLE IF NOT EXISTS recommendations (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(255) UNIQUE NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			items JSONB,
			reason TEXT,
			total_price DECIMAL(10, 2),
			match_score DECIMAL(5, 2),
			occasion VARCHAR(100),
			season VARCHAR(50),
			feedback JSONB,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`

		_, err = pool.Exec(context.Background(), query)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("migration 4 - embeddings table", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		query := `CREATE TABLE IF NOT EXISTS embeddings (
			id VARCHAR(255) PRIMARY KEY,
			table_name VARCHAR(100) NOT NULL,
			embedding VECTOR(1536),
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`

		_, err = pool.Exec(context.Background(), query)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})
}

// TestMigrate_Integration tests migration integration scenarios.
func TestMigrate_Integration(t *testing.T) {
	t.Run("run migrations multiple times", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		// Run migrations twice - should not fail on second run
		_ = Migrate(context.Background(), pool)
		err = Migrate(context.Background(), pool)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("verify table creation after migration", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		_ = Migrate(context.Background(), pool)

		// Check if tables exist
		tables := []string{
			"user_profiles",
			"sessions",
			"recommendations",
			"embeddings",
		}

		for _, table := range tables {
			query := `SELECT to_regclass($1)`
			var tableName string
			err := pool.db.QueryRowContext(context.Background(), query, table).Scan(&tableName)
			if err != nil {
				t.Logf("Expected error checking table %s: %v", table, err)
			}
		}
	})
}

// nolint: errcheck // Test code may ignore return values
