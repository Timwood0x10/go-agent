package resources

import (
	"context"
	"fmt"

	"goagent/api/core"
)

// KnowledgeSearch searches the knowledge base for relevant information.
type KnowledgeSearch struct {
	*BaseTool
	searcher KnowledgeSearcher
}

// KnowledgeSearcher defines the interface for searching knowledge base.
type KnowledgeSearcher interface {
	Search(ctx context.Context, tenantID, query string) ([]*core.RetrievalResult, error)
}

// NewKnowledgeSearch creates a new KnowledgeSearch tool.
func NewKnowledgeSearch(searcher KnowledgeSearcher) *KnowledgeSearch {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"tenant_id": {
				Type:        "string",
				Description: "Tenant identifier for multi-tenant isolation",
			},
			"query": {
				Type:        "string",
				Description: "Search query text",
			},
			"top_k": {
				Type:        "integer",
				Description: "Number of top results to return (1-50)",
				Default:     10,
			},
			"min_score": {
				Type:        "number",
				Description: "Minimum similarity score threshold (0.0-1.0)",
				Default:     0.4,
			},
		},
		Required: []string{"tenant_id", "query"},
	}

	ks := &KnowledgeSearch{
		searcher: searcher,
	}
	ks.BaseTool = NewBaseTool("knowledge_search", "Search knowledge base for relevant information", params)

	return ks
}

// Execute performs the knowledge base search.
func (t *KnowledgeSearch) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	tenantID, ok := params["tenant_id"].(string)
	if !ok || tenantID == "" {
		return NewErrorResult("tenant_id is required"), nil
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return NewErrorResult("query is required"), nil
	}

	// Note: For now, we use the simple search. Advanced filtering (top_k, min_score)
	// can be implemented if the searcher supports it.
	results, err := t.searcher.Search(ctx, tenantID, query)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("search failed: %v", err)), nil
	}

	// Format results
	items := make([]map[string]interface{}, len(results))
	for i, result := range results {
		items[i] = map[string]interface{}{
			"id":       result.ID,
			"content":  result.Content,
			"source":   result.Source,
			"score":    result.Score,
			"metadata": result.Metadata,
		}
	}

	return NewResult(true, map[string]interface{}{
		"results": items,
		"total":   len(results),
		"query":   query,
	}), nil
}

// KnowledgeUpdate updates an existing knowledge item to correct errors or outdated information.
type KnowledgeUpdate struct {
	*BaseTool
	service core.RetrievalService
}

// NewKnowledgeUpdate creates a new KnowledgeUpdate tool.
func NewKnowledgeUpdate(service core.RetrievalService) *KnowledgeUpdate {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"tenant_id": {
				Type:        "string",
				Description: "Tenant identifier for multi-tenant isolation",
			},
			"item_id": {
				Type:        "string",
				Description: "Knowledge item ID to update",
			},
			"content": {
				Type:        "string",
				Description: "Corrected content",
			},
			"source": {
				Type:        "string",
				Description: "Source of the information",
			},
			"category": {
				Type:        "string",
				Description: "Category of the knowledge",
			},
			"tags": {
				Type:        "array",
				Description: "Tags for categorization",
			},
			"reason": {
				Type:        "string",
				Description: "Reason for the update (e.g., 'correction', 'outdated', 'expansion')",
			},
		},
		Required: []string{"tenant_id", "item_id", "content"},
	}

	ku := &KnowledgeUpdate{
		service: service,
	}
	ku.BaseTool = NewBaseTool("knowledge_update", "Update knowledge base item to correct errors or outdated information", params)

	return ku
}

// Execute updates a knowledge item.
func (t *KnowledgeUpdate) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	tenantID, ok := params["tenant_id"].(string)
	if !ok || tenantID == "" {
		return NewErrorResult("tenant_id is required"), nil
	}

	itemID, ok := params["item_id"].(string)
	if !ok || itemID == "" {
		return NewErrorResult("item_id is required"), nil
	}

	content, ok := params["content"].(string)
	if !ok || content == "" {
		return NewErrorResult("content is required"), nil
	}

	// First, retrieve the existing item to preserve other fields
	existing, err := t.service.GetKnowledge(ctx, tenantID, itemID)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to get existing item: %v", err)), nil
	}

	// Update fields
	existing.Content = content
	if source, ok := params["source"].(string); ok {
		existing.Source = source
	}
	if category, ok := params["category"].(string); ok {
		existing.Category = category
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		tagStrings := make([]string, len(tags))
		for i, tag := range tags {
			if s, ok := tag.(string); ok {
				tagStrings[i] = s
			}
		}
		existing.Tags = tagStrings
	}

	// Add update reason to metadata
	reason := getString(params, "reason")
	if reason != "" {
		if existing.Metadata == nil {
			existing.Metadata = make(core.Metadata)
		}
		existing.Metadata["update_reason"] = reason
	}

	// Perform update
	updated, err := t.service.UpdateKnowledge(ctx, tenantID, existing)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("update failed: %v", err)), nil
	}

	return NewResult(true, map[string]interface{}{
		"item_id":    updated.ID,
		"content":    updated.Content,
		"updated_at": updated.UpdatedAt,
		"success":    true,
	}), nil
}

// KnowledgeAdd adds new knowledge to the knowledge base.
type KnowledgeAdd struct {
	*BaseTool
	service core.RetrievalService
}

// NewKnowledgeAdd creates a new KnowledgeAdd tool.
func NewKnowledgeAdd(service core.RetrievalService) *KnowledgeAdd {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"tenant_id": {
				Type:        "string",
				Description: "Tenant identifier for multi-tenant isolation",
			},
			"content": {
				Type:        "string",
				Description: "Knowledge content to add",
			},
			"source": {
				Type:        "string",
				Description: "Source of the information",
			},
			"category": {
				Type:        "string",
				Description: "Category of the knowledge",
			},
			"tags": {
				Type:        "array",
				Description: "Tags for categorization",
			},
		},
		Required: []string{"tenant_id", "content"},
	}

	ka := &KnowledgeAdd{
		service: service,
	}
	ka.BaseTool = NewBaseTool("knowledge_add", "Add new knowledge to the knowledge base", params)

	return ka
}

// Execute adds a new knowledge item.
func (t *KnowledgeAdd) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	tenantID, ok := params["tenant_id"].(string)
	if !ok || tenantID == "" {
		return NewErrorResult("tenant_id is required"), nil
	}

	content, ok := params["content"].(string)
	if !ok || content == "" {
		return NewErrorResult("content is required"), nil
	}

	item := &core.KnowledgeItem{
		TenantID: tenantID,
		Content:  content,
	}

	if source, ok := params["source"].(string); ok {
		item.Source = source
	}
	if category, ok := params["category"].(string); ok {
		item.Category = category
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		tagStrings := make([]string, len(tags))
		for i, tag := range tags {
			if s, ok := tag.(string); ok {
				tagStrings[i] = s
			}
		}
		item.Tags = tagStrings
	}

	created, err := t.service.AddKnowledge(ctx, item)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("add failed: %v", err)), nil
	}

	return NewResult(true, map[string]interface{}{
		"item_id":    created.ID,
		"content":    created.Content,
		"created_at": created.CreatedAt,
		"success":    true,
	}), nil
}

// KnowledgeDelete removes a knowledge item from the knowledge base.
type KnowledgeDelete struct {
	*BaseTool
	service core.RetrievalService
}

// NewKnowledgeDelete creates a new KnowledgeDelete tool.
func NewKnowledgeDelete(service core.RetrievalService) *KnowledgeDelete {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"tenant_id": {
				Type:        "string",
				Description: "Tenant identifier for multi-tenant isolation",
			},
			"item_id": {
				Type:        "string",
				Description: "Knowledge item ID to delete",
			},
			"reason": {
				Type:        "string",
				Description: "Reason for deletion (e.g., 'incorrect', 'outdated', 'duplicate')",
			},
		},
		Required: []string{"tenant_id", "item_id"},
	}

	kd := &KnowledgeDelete{
		service: service,
	}
	kd.BaseTool = NewBaseTool("knowledge_delete", "Remove a knowledge item from the knowledge base", params)

	return kd
}

// Execute deletes a knowledge item.
func (t *KnowledgeDelete) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	tenantID, ok := params["tenant_id"].(string)
	if !ok || tenantID == "" {
		return NewErrorResult("tenant_id is required"), nil
	}

	itemID, ok := params["item_id"].(string)
	if !ok || itemID == "" {
		return NewErrorResult("item_id is required"), nil
	}

	reason := getString(params, "reason")

	err := t.service.DeleteKnowledge(ctx, tenantID, itemID)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("delete failed: %v", err)), nil
	}

	return NewResult(true, map[string]interface{}{
		"item_id": itemID,
		"reason":  reason,
		"success": true,
	}), nil
}
