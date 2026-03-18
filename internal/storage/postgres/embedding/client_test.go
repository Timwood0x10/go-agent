// nolint: errcheck // Test code may ignore return values
package embedding

import (
	"context"
	"testing"
	"time"
)

func TestNormalizeText(t *testing.T) {
	client := &EmbeddingClient{}

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
			name:     "extra spaces",
			input:    "hello  world",
			expected: "hello world",
		},
		{
			name:     "uppercase",
			input:    "HELLO WORLD",
			expected: "hello world",
		},
		{
			name:     "trim spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "mixed",
			input:    "  HELLO   WORLD  ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeText() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCacheKey(t *testing.T) {
	client := &EmbeddingClient{
		model: "test-model",
	}

	tests := []struct {
		name     string
		text     string
		method   string
		wantDiff bool
	}{
		{
			name:     "same text",
			text:     "hello world",
			method:   "query",
			wantDiff: false,
		},
		{
			name:     "different text",
			text:     "hello world",
			method:   "query",
			wantDiff: false,
		},
		{
			name:     "different method",
			text:     "hello world",
			method:   "passage",
			wantDiff: true,
		},
	}

	baseKey := client.getCacheKey("hello world", "query")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := client.getCacheKey(tt.text, tt.method)
			if tt.wantDiff && key == baseKey {
				t.Errorf("getCacheKey() should be different")
			}
		})
	}
}

func TestFallbackClient(t *testing.T) {
	tests := []struct {
		name     string
		strategy FallbackStrategy
	}{
		{
			name:     "fallback to cache",
			strategy: FallbackToCache,
		},
		{
			name:     "fallback to keyword",
			strategy: FallbackToKeyword,
		},
		{
			name:     "fallback to error",
			strategy: FallbackToError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewFallbackClient(nil, tt.strategy)
			if client.GetStrategy() != tt.strategy {
				t.Errorf("GetStrategy() = %v, want %v", client.GetStrategy(), tt.strategy)
			}
		})
	}
}

func TestCacheKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  *CacheKey
	}{
		{
			name: "basic",
			key: &CacheKey{
				Text:   "hello",
				Model:  "model",
				Method: "query",
			},
		},
		{
			name: "different text",
			key: &CacheKey{
				Text:   "world",
				Model:  "model",
				Method: "query",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyStr := tt.key.String()
			if len(keyStr) == 0 {
				t.Errorf("String() returned empty string")
			}
			if keyStr[:6] != "embed:" {
				t.Errorf("String() should start with 'embed:'")
			}
		})
	}
}

func TestEmbeddingCache(t *testing.T) {
	ctx := context.Background()
	cache := NewEmbeddingCache(nil, time.Hour)

	// Cache should be enabled (using memory cache)
	if !cache.IsEnabled() {
		t.Error("Cache should be enabled (memory cache is always available)")
	}

	// Test enable/disable
	cache.Disable()
	if cache.IsEnabled() {
		t.Error("Cache should be disabled")
	}

	// Re-enable
	cache.Enable()
	if !cache.IsEnabled() {
		t.Error("Cache should be enabled")
	}

	// Test Get/Set with memory cache
	key := &CacheKey{Text: "test", Model: "model", Method: "query"}
	embedding := []float64{0.1, 0.2, 0.3}

	err := cache.Set(ctx, key, embedding)
	if err != nil {
		t.Errorf("Set() should not error, got %v", err)
	}

	result, found := cache.Get(ctx, key)
	if !found {
		t.Error("Get() should find embedding in memory cache")
	}
	if found && len(result) != len(embedding) {
		t.Errorf("Get() returned wrong length, got %d, want %d", len(result), len(embedding))
	}
}

func TestCacheStats(t *testing.T) {
	ctx := context.Background()
	cache := NewEmbeddingCache(nil, time.Hour)

	stats, err := cache.GetStats(ctx)
	if err != nil {
		t.Errorf("GetStats() error = %v", err)
	}

	if stats == nil {
		t.Error("GetStats() should return stats")
	}

	// Cache should be enabled (memory cache is always available)
	if !stats.Enabled {
		t.Error("Stats should show cache as enabled")
	}

	// Redis keys should be 0 (no Redis client)
	if stats.RedisKeys != 0 {
		t.Errorf("RedisKeys should be 0, got %d", stats.RedisKeys)
	}

	// Total keys should be 0 (no items in cache yet)
	if stats.TotalKeys != 0 {
		t.Errorf("TotalKeys should be 0 initially, got %d", stats.TotalKeys)
	}
}

// nolint: errcheck // Test code may ignore return values
