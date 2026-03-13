package resources

import (
	"context"
	"strconv"

	"goagent/internal/core/models"
)

// FashionSearch searches for fashion items.
type FashionSearch struct {
	*BaseTool
	searcher FashionSearcher
}

// FashionSearcher defines the interface for fashion searching.
type FashionSearcher interface {
	Search(ctx context.Context, query string, filters *FashionFilters) ([]*FashionItem, error)
}

// FashionFilters holds search filters.
type FashionFilters struct {
	Category string
	Style    []string
	PriceMin float64
	PriceMax float64
	Colors   []string
	Brands   []string
	Occasion string
	Season   string
}

// FashionItem represents a fashion item.
type FashionItem struct {
	ItemID   string                 `json:"item_id"`
	Name     string                 `json:"name"`
	Brand    string                 `json:"brand"`
	Category string                 `json:"category"`
	Price    float64                `json:"price"`
	URL      string                 `json:"url"`
	ImageURL string                 `json:"image_url"`
	Style    []string               `json:"style"`
	Colors   []string               `json:"colors"`
	Occasion string                 `json:"occasion"`
	Season   string                 `json:"season"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewFashionSearch creates a new FashionSearch tool.
func NewFashionSearch(searcher FashionSearcher) *FashionSearch {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"query": {
				Type:        "string",
				Description: "Search query",
			},
			"category": {
				Type:        "string",
				Description: "Category filter (top, bottom, dress, outerwear, shoes, accessory)",
			},
			"style": {
				Type:        "array",
				Description: "Style tags",
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
	fs.BaseTool = NewBaseTool("fashion_search", "Search for fashion items", params)

	return fs
}

// Execute performs the fashion search.
func (t *FashionSearch) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return NewErrorResult("query is required"), nil
	}

	filters := &FashionFilters{
		Category: getString(params, "category"),
		Style:    getStringSlice(params, "style"),
		PriceMin: getFloat(params, "price_min"),
		PriceMax: getFloat(params, "price_max"),
		Colors:   getStringSlice(params, "colors"),
	}

	limit := getInt(params, "limit", 10)

	items, err := t.searcher.Search(ctx, query, filters)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	if len(items) > limit {
		items = items[:limit]
	}

	// Convert to models.RecommendItem
	recommendations := make([]*models.RecommendItem, len(items))
	for i, item := range items {
		recommendations[i] = &models.RecommendItem{
			ItemID:      item.ItemID,
			Category:    item.Category,
			Name:        item.Name,
			Brand:       item.Brand,
			Price:       item.Price,
			URL:         item.URL,
			ImageURL:    item.ImageURL,
			Style:       parseStyleTags(item.Style),
			Colors:      item.Colors,
			Description: item.Name + " - " + item.Brand,
			MatchReason: "Matches: " + query,
		}
	}

	return NewResult(true, map[string]interface{}{
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

func parseStyleTags(tags []string) []models.StyleTag {
	result := make([]models.StyleTag, len(tags))
	for i, tag := range tags {
		result[i] = models.StyleTag(tag)
	}
	return result
}
