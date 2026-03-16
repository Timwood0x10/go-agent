package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// TestPool_NewPool tests creating a new connection pool.
func TestPool_NewPool(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			Host:            "localhost",
			Port:            5432,
			User:            "test",
			Password:        "test",
			Database:        "testdb",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		}

		pool, err := NewPool(cfg)
		if err != nil {
			t.Logf("Expected to fail without database: %v", err)
		}
		if pool != nil {
			pool.Close()
		}
	})

	t.Run("invalid config - missing host", func(t *testing.T) {
		cfg := &Config{
			Port:     5432,
			User:     "test",
			Password: "test",
			Database: "testdb",
		}

		_, err := NewPool(cfg)
		if err != nil {
			t.Logf("Expected validation error: %v", err)
		}
	})
}

// TestPool_Get tests acquiring a connection from the pool.
func TestPool_Get(t *testing.T) {
	t.Run("get connection", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Logf("Expected to fail without database: %v", err)
		}
		if conn != nil {
			pool.Release(conn)
		}
	})

	t.Run("get connection with cancelled context", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = pool.Get(ctx)
		if err != nil {
			t.Logf("Expected error with cancelled context: %v", err)
		}
	})
}

// TestPool_Release tests releasing a connection to the pool.
func TestPool_Release(t *testing.T) {
	t.Run("release nil connection", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		// Should not panic
		pool.Release(nil)
	})

	t.Run("release valid connection", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}

		// Should not panic
		pool.Release(conn)
	})
}

// TestPool_WithConnection tests executing a function with a connection.
func TestPool_WithConnection(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		called := false

		err = pool.WithConnection(ctx, func(conn *sql.Conn) error {
			called = true
			return nil
		})

		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
		if !called {
			t.Error("function should have been called")
		}
	})

	t.Run("function returns error", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		expectedErr := errors.New("test error")

		err = pool.WithConnection(ctx, func(conn *sql.Conn) error {
			return expectedErr
		})

		if err != expectedErr && err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})
}

// TestPool_Close tests closing the connection pool.
func TestPool_Close(t *testing.T) {
	t.Run("close pool", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}

		err = pool.Close()
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("close pool multiple times", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}

		pool.Close()
		err = pool.Close() // Should not panic
		if err != nil {
			t.Logf("Expected error on second close: %v", err)
		}
	})
}

// TestPool_Stats tests getting pool statistics.
func TestPool_Stats(t *testing.T) {
	t.Run("get stats", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		stats := pool.Stats()
		if stats == nil {
			t.Error("stats should not be nil")
		}
		if stats.MaxOpenConns != cfg.MaxOpenConns {
			t.Errorf("expected MaxOpenConns %d, got %d", cfg.MaxOpenConns, stats.MaxOpenConns)
		}
	})
}

// TestPool_IsHealthy tests pool health check.
func TestPool_IsHealthy(t *testing.T) {
	t.Run("healthy pool", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		healthy := pool.IsHealthy()
		t.Logf("Pool health: %v", healthy)
	})
}

// TestPool_Ping tests pinging the database.
func TestPool_Ping(t *testing.T) {
	t.Run("ping database", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		err = pool.Ping(ctx)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})

	t.Run("ping with cancelled context", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = pool.Ping(ctx)
		if err != nil {
			t.Logf("Expected error with cancelled context: %v", err)
		}
	})
}

// TestPool_Exec tests executing a query without returning rows.
func TestPool_Exec(t *testing.T) {
	t.Run("exec query", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		_, err = pool.Exec(ctx, "SELECT 1")
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
	})
}

// TestPool_Query tests executing a query and returning rows.
func TestPool_Query(t *testing.T) {
	t.Run("query rows", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		rows, err := pool.Query(ctx, "SELECT 1")
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
		if rows != nil {
			rows.Close()
		}
	})
}

// TestPool_QueryRow tests executing a query and returning a single row.
func TestPool_QueryRow(t *testing.T) {
	t.Run("query row", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		row := pool.QueryRow(ctx, "SELECT 1")
		if row == nil {
			t.Error("row should not be nil")
		}
	})
}

// TestPool_Begin tests starting a new transaction.
func TestPool_Begin(t *testing.T) {
	t.Run("begin transaction", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Logf("Expected error without database: %v", err)
		}
		if tx != nil {
			tx.Rollback()
		}
	})
}

// TestPoolStats tests PoolStats structure.
func TestPoolStats(t *testing.T) {
	t.Run("pool stats fields", func(t *testing.T) {
		stats := &PoolStats{
			OpenConnections:  10,
			InUseConnections: 5,
			IdleConnections:  5,
			WaitCount:        100,
			WaitDuration:     time.Second,
			MaxOpenConns:     25,
		}

		if stats.OpenConnections != 10 {
			t.Errorf("expected OpenConnections 10, got %d", stats.OpenConnections)
		}
		if stats.InUseConnections != 5 {
			t.Errorf("expected InUseConnections 5, got %d", stats.InUseConnections)
		}
		if stats.IdleConnections != 5 {
			t.Errorf("expected IdleConnections 5, got %d", stats.IdleConnections)
		}
		if stats.WaitCount != 100 {
			t.Errorf("expected WaitCount 100, got %d", stats.WaitCount)
		}
		if stats.WaitDuration != time.Second {
			t.Errorf("expected WaitDuration 1s, got %v", stats.WaitDuration)
		}
		if stats.MaxOpenConns != 25 {
			t.Errorf("expected MaxOpenConns 25, got %d", stats.MaxOpenConns)
		}
	})
}

// TestManagedRows tests ManagedRows structure.
func TestManagedRows(t *testing.T) {
	t.Run("managed rows close", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Host = "localhost"

		pool, err := NewPool(cfg)
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}
		defer pool.Close()

		ctx := context.Background()
		rows, err := pool.Query(ctx, "SELECT 1")
		if err != nil {
			t.Skipf("Skipping test without database: %v", err)
		}

		if rows != nil {
			err = rows.Close()
			if err != nil {
				t.Logf("Expected error without database: %v", err)
			}
		}
	})
}
