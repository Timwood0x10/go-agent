package resources

import (
	"context"
	"fmt"

	"goagent/internal/storage/postgres/repositories"
)

// DistilledMemorySearch searches distilled memories from the database.
type DistilledMemorySearch struct {
	*BaseTool
	repo *repositories.DistilledMemoryRepository
}

// NewDistilledMemorySearch creates a new DistilledMemorySearch tool.
func NewDistilledMemorySearch(repo *repositories.DistilledMemoryRepository) *DistilledMemorySearch {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"tenant_id": {
				Type:        "string",
				Description: "Tenant identifier for multi-tenant isolation",
			},
			"user_id": {
				Type:        "string",
				Description: "User identifier to search for",
			},
			"query": {
				Type:        "string",
				Description: "Search query for vector similarity search",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 5)",
				Default:     5,
			},
		},
		Required: []string{"tenant_id"},
	}

	dms := &DistilledMemorySearch{
		repo: repo,
	}
	dms.BaseTool = NewBaseTool("distilled_memory_search", "Search distilled memories and user preferences from database", params)

	return dms
}

// Execute searches distilled memories.
func (t *DistilledMemorySearch) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	tenantID, ok := params["tenant_id"].(string)
	if !ok || tenantID == "" {
		return NewErrorResult("tenant_id is required"), nil
	}

	// If user_id is provided, search by user_id
	if userID, ok := params["user_id"].(string); ok && userID != "" {
		memories, err := t.repo.GetByUserID(ctx, tenantID, userID, 10)
		if err != nil {
			return NewErrorResult(fmt.Sprintf("failed to get memories by user: %v", err)), nil
		}

		// Format results
		items := make([]map[string]interface{}, len(memories))
		for i, mem := range memories {
			items[i] = map[string]interface{}{
				"id":          mem.ID,
				"user_id":     mem.UserID,
				"session_id":  mem.SessionID,
				"content":     mem.Content,
				"memory_type": mem.MemoryType,
				"importance":  mem.Importance,
				"created_at":  mem.CreatedAt,
				"expires_at":  mem.ExpiresAt,
			}
		}

		return NewResult(true, map[string]interface{}{
			"user_id":  userID,
			"memories": items,
			"total":    len(items),
		}), nil
	}

	// Otherwise, perform vector search
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return NewErrorResult("either user_id or query is required"), nil
	}

	// Note: We would need embedding generation here for vector search
	// For now, return empty results
	return NewResult(true, map[string]interface{}{
		"query":    query,
		"memories": []map[string]interface{}{},
		"total":    0,
		"message":  "Vector search requires embedding generation",
	}), nil
}
