package builtin

import (
	"context"
	"strconv"

	"goagent/internal/core/models"
	"goagent/internal/tools/resources/base"
	"goagent/internal/tools/resources/core"
	"goagent/internal/tools/resources/types"
)

// FashionSearch searches for fashion items.
type FashionSearch struct {
	*base.BaseTool
	searcher FashionSearcher
}

// FashionSearcher defines the interface for fashion searching.
type FashionSearcher interface {
	Search(ctx context.Context, query string, filters *types.FashionFilters) ([]*types.FashionItem, error)
}

// NewFashionSearch creates a new FashionSearch tool.
func NewFashionSearch(searcher FashionSearcher) *FashionSearch {
	params := &core.ParameterSchema{
		Type: "object",
		Properties: map[string]*core.Parameter{
			"query": {
				Type:        "string",
				Description: "Search query",
			},
			"category": {
				Type:        "string",
				Description: "Category filter (top, bottom, dress, outerwear, shoes, accessory)",
			},
			"agent_preferences": {
				Type:        "array",
				Description: "Agent preferences",
			},
			"price_min": {
				Type:        "number",
				Description: "Minimum price",
			},
			"price_max": {
				Type:        "number",
				Description: "Maximum price",
			},
			"colors": {
				Type:        "array",
				Description: "Preferred colors",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum results",
				Default:     10,
			},
		},
		Required: []string{"query"},
	}

	fs := &FashionSearch{
		searcher: searcher,
	}
	fs.BaseTool = base.NewBaseTool("fashion_search", "Search for fashion items", params)

	return fs
}

// Execute performs the fashion search.
func (t *FashionSearch) Execute(ctx context.Context, params map[string]interface{}) (core.Result, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return core.NewErrorResult("query is required"), nil
	}

	filters := &types.FashionFilters{
		Category:         getString(params, "category"),
		AgentPreferences: getStringSlice(params, "agent_preferences"),
		PriceMin:         getFloat(params, "price_min"),
		PriceMax:         getFloat(params, "price_max"),
		Colors:           getStringSlice(params, "colors"),
	}

	limit := getInt(params, "limit", 10)

	items, err := t.searcher.Search(ctx, query, filters)
	if err != nil {
		return core.NewErrorResult(err.Error()), nil
	}

	if len(items) > limit {
		items = items[:limit]
	}

	// Convert to models.RecommendItem
	recommendations := make([]*models.RecommendItem, len(items))
	for i, item := range items {
		recommendations[i] = &models.RecommendItem{
			ItemID:           item.ItemID,
			Category:         item.Category,
			Name:             item.Name,
			Brand:            item.Brand,
			Price:            item.Price,
			URL:              item.URL,
			ImageURL:         item.ImageURL,
			AgentPreferences: parseAgentPreferences(item.AgentPreferences),
			Colors:           item.Colors,
			Description:      item.Name + " - " + item.Brand,
			MatchReason:      "Matches: " + query,
		}
	}

	return core.NewResult(true, map[string]interface{}{
		"items":         recommendations,
		"total_results": len(items),
		"query":         query,
	}), nil
}

// Helper functions.
func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func getStringSlice(params map[string]interface{}, key string) []string {
	if v, ok := params[key].([]interface{}); ok {
		result := make([]string, len(v))
		for i, val := range v {
			if s, ok := val.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	return nil
}

func getFloat(params map[string]interface{}, key string) float64 {
	switch v := params[key].(type) {
	case float64:
		return v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

func getInt(params map[string]interface{}, key string, defaultVal int) int {
	switch v := params[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func parseAgentPreferences(tags []string) []models.StyleTag {
	result := make([]models.StyleTag, len(tags))
	for i, tag := range tags {
		result[i] = models.StyleTag(tag)
	}
	return result
}