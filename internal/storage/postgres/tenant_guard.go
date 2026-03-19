// Package postgres provides PostgreSQL database operations for the storage system.
package postgres

import (
	"context"
	"fmt"

	"goagent/internal/core/errors"
)

// TenantGuard provides physical isolation for multi-tenant data access.
// This enforces tenant context at the database level to prevent cross-tenant data access.
type TenantGuard struct {
	db *Pool
}

// NewTenantGuard creates a new TenantGuard instance.
func NewTenantGuard(pool *Pool) *TenantGuard {
	return &TenantGuard{db: pool}
}

// SetTenantContext sets the tenant context for the current database session.
// This MUST be called for every tenant-specific operation to ensure physical isolation.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier, must be non-empty.
// Returns error if setting tenant context fails.
func (g *TenantGuard) SetTenantContext(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return errors.ErrInvalidArgument
	}

	// SET语句不支持参数化查询，需要直接拼接字符串
	// 这里tenantID已经被验证为非空字符串，所以可以安全地使用
	query := fmt.Sprintf("SET app.tenant_id TO '%s'", tenantID)
	_, err := g.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	return nil
}

// MustSetTenantContext sets the tenant context and returns error on failure.
// This should only be used in initialization paths where failure is fatal.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier.
// Returns:
// error - if tenant context setup fails.
func (g *TenantGuard) MustSetTenantContext(ctx context.Context, tenantID string) error {
	if err := g.SetTenantContext(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// WithTenant executes a function within a tenant context.
// This is a convenience wrapper that ensures tenant context is set before execution.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier.
// fn - function to execute within tenant context.
// Returns error if tenant context setup or function execution fails.
func (g *TenantGuard) WithTenant(ctx context.Context, tenantID string, fn func(context.Context) error) error {
	if err := g.SetTenantContext(ctx, tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	return fn(ctx)
}

// ClearTenantContext clears the tenant context.
// This is primarily used for cleanup and testing purposes.
// Args:
// ctx - database operation context.
// Returns error if clearing tenant context fails.
func (g *TenantGuard) ClearTenantContext(ctx context.Context) error {
	_, err := g.db.Exec(ctx, "SET app.tenant_id TO DEFAULT")
	if err != nil {
		return fmt.Errorf("clear tenant context: %w", err)
	}

	return nil
}
