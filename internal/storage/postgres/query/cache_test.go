package query

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestQueryCache(t *testing.T) {
	ctx := context.Background()
	cache := NewQueryCache(nil, time.Hour)

	// Test initial state
	if !cache.IsEnabled() {
		t.Error("Cache should be enabled")
	}

	// Test Get/Set
	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		Filters: map[string]interface{}{
			"type": "knowledge",
		},
		TopK: 10,
	}

	results := []*SearchResult{
		{
			ID:      "1",
			Content: "Result 1",
			Source:  "knowledge",
			Score:   0.9,
			Metadata: map[string]interface{}{
				"key": "value",
			},
		},
		{
			ID:      "2",
			Content: "Result 2",
			Source:  "knowledge",
			Score:   0.8,
		},
	}

	// Store results
	err := cache.Set(ctx, req, results)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Retrieve results
	cached, err := cache.Get(ctx, req)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if cached == nil {
		t.Error("Get() should return results")
	}

	if len(cached) != len(results) {
		t.Errorf("Get() returned wrong length, got %d, want %d", len(cached), len(results))
	}

	// Test stats
	stats := cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	hitRate := stats.HitRate()
	if hitRate != 1.0 {
		t.Errorf("Expected hit rate 1.0, got %f", hitRate)
	}
}

func TestQueryCacheMiss(t *testing.T) {
	ctx := context.Background()
	cache := NewQueryCache(nil, time.Hour)

	req := &SearchRequest{
		Query:    "non-existent query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	// Try to get non-existent result
	_, err := cache.Get(ctx, req)
	if err != ErrQueryNotFound {
		t.Errorf("Expected ErrQueryNotFound, got %v", err)
	}

	// Test stats
	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	hitRate := stats.HitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0, got %f", hitRate)
	}
}

func TestQueryCacheDelete(t *testing.T) {
	ctx := context.Background()
	cache := NewQueryCache(nil, time.Hour)

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	results := []*SearchResult{
		{
			ID:      "1",
			Content: "Result 1",
			Score:   0.9,
		},
	}

	// Store results
	err := cache.Set(ctx, req, results)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Delete results
	err = cache.Delete(ctx, req)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = cache.Get(ctx, req)
	if err != ErrQueryNotFound {
		t.Errorf("Expected ErrQueryNotFound after deletion, got %v", err)
	}
}

func TestQueryCacheClear(t *testing.T) {
	ctx := context.Background()
	cache := NewQueryCache(nil, time.Hour)

	// Add multiple items
	for i := 0; i < 5; i++ {
		req := &SearchRequest{
			Query:    fmt.Sprintf("query %d", i),
			TenantID: "tenant-1",
			TopK:     10,
		}

		results := []*SearchResult{
			{
				ID:      fmt.Sprintf("result %d", i),
				Content: fmt.Sprintf("Content %d", i),
				Score:   float64(i) / 10,
			},
		}

		err := cache.Set(ctx, req, results)
		if err != nil {
			t.Errorf("Set() error = %v", err)
		}
	}

	// Clear cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// Verify all items are cleared
	for i := 0; i < 5; i++ {
		req := &SearchRequest{
			Query:    fmt.Sprintf("query %d", i),
			TenantID: "tenant-1",
			TopK:     10,
		}

		_, err := cache.Get(ctx, req)
		if err != ErrQueryNotFound {
			t.Errorf("Expected ErrQueryNotFound after clear, got %v", err)
		}
	}
}

func TestQueryCacheDisable(t *testing.T) {
	ctx := context.Background()
	cache := NewQueryCache(nil, time.Hour)

	req := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	results := []*SearchResult{
		{
			ID:      "1",
			Content: "Result 1",
			Score:   0.9,
		},
	}

	// Disable cache
	cache.Disable()
	if cache.IsEnabled() {
		t.Error("Cache should be disabled")
	}

	// Try to store (should succeed but not cache)
	err := cache.Set(ctx, req, results)
	if err != nil {
		t.Errorf("Set() should succeed even when disabled")
	}

	// Try to get (should return not found)
	_, err = cache.Get(ctx, req)
	if err != ErrQueryNotFound {
		t.Errorf("Expected ErrQueryNotFound when disabled, got %v", err)
	}

	// Re-enable cache
	cache.Enable()
	if !cache.IsEnabled() {
		t.Error("Cache should be enabled")
	}
}

func TestQueryCacheKeyGeneration(t *testing.T) {
	cache := NewQueryCache(nil, time.Hour)

	req1 := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	req2 := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	key1 := cache.getCacheKey(req1)
	key2 := cache.getCacheKey(req2)

	if key1 != key2 {
		t.Error("Same requests should generate same cache key")
	}

	// Different tenant
	req3 := &SearchRequest{
		Query:    "test query",
		TenantID: "tenant-2",
		TopK:     10,
	}

	key3 := cache.getCacheKey(req3)
	if key3 == key1 {
		t.Error("Different tenants should generate different cache keys")
	}

	// Different query
	req4 := &SearchRequest{
		Query:    "different query",
		TenantID: "tenant-1",
		TopK:     10,
	}

	key4 := cache.getCacheKey(req4)
	if key4 == key1 {
		t.Error("Different queries should generate different cache keys")
	}
}

func TestMemoryQueryCache(t *testing.T) {
	cache := NewMemoryQueryCache()

	key := "test-key"
	results := []*SearchResult{
		{
			ID:      "1",
			Content: "Result 1",
			Score:   0.9,
		},
	}

	// Test Set/Get
	cache.Set(key, results, time.Hour)

	cached, found := cache.Get(key)
	if !found {
		t.Error("Memory cache should find stored item")
	}

	if len(cached) != len(results) {
		t.Errorf("Memory cache returned wrong length, got %d, want %d", len(cached), len(results))
	}

	// Test Delete
	cache.Delete(key)

	_, found = cache.Get(key)
	if found {
		t.Error("Memory cache should not find deleted item")
	}

	// Test Clear
	cache.Set("key1", results, time.Hour)
	cache.Set("key2", results, time.Hour)
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Memory cache should be empty after clear, got %d items", cache.Len())
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "trim spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "mixed",
			input:    "  HELLO WORLD  ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeText() = %v, want %v", result, tt.expected)
			}
		})
	}
}
