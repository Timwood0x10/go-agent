package leader

import (
	"context"
	"sort"

	"styleagent/internal/core/models"
)

// resultAggregator aggregates results from sub-agents.
type resultAggregator struct {
	enableDedupe bool
	maxItems     int
}

// NewResultAggregator creates a new ResultAggregator.
func NewResultAggregator(enableDedupe bool, maxItems int) ResultAggregator {
	if maxItems <= 0 {
		maxItems = 20
	}
	return &resultAggregator{
		enableDedupe: enableDedupe,
		maxItems:     maxItems,
	}
}

// Aggregate combines results from all sub-agents.
func (a *resultAggregator) Aggregate(ctx context.Context, results []*models.TaskResult) (*models.RecommendResult, error) {
	// Collect all items
	allItems := make([]*models.RecommendItem, 0)
	successCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
			allItems = append(allItems, result.Items...)
		}
	}

	// Deduplicate if enabled
	if a.enableDedupe {
		allItems = deduplicateItems(allItems)
	}

	// Sort by price (descending) as a simple proxy for quality
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Price > allItems[j].Price
	})

	// Limit items
	if len(allItems) > a.maxItems {
		allItems = allItems[:a.maxItems]
	}

	result := models.NewRecommendResult("", "")
	result.Items = allItems
	for _, item := range allItems {
		result.TotalPrice += item.Price
	}

	if len(results) > 0 {
		result.MatchScore = float64(successCount) / float64(len(results))
	}

	return result, nil
}

func deduplicateItems(items []*models.RecommendItem) []*models.RecommendItem {
	seen := make(map[string]bool)
	result := make([]*models.RecommendItem, 0)

	for _, item := range items {
		if !seen[item.ItemID] {
			seen[item.ItemID] = true
			result = append(result, item)
		}
	}

	return result
}
