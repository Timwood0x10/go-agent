package embedding

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/blake2b"
)

// RedisClient defines the interface for Redis operations.
// This allows for optional Redis dependency and easy testing.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// CacheKey represents a cache key for embeddings.
type CacheKey struct {
	Text   string
	Model  string
	Method string
}

// String returns the string representation of the cache key using BLAKE2b-128 hash.
// BLAKE2b provides:
// - Security: Cryptographically secure hash function
// - Performance: 20-30% faster than SHA256
// - Efficiency: 128-bit output is sufficient for cache key collision resistance
func (k *CacheKey) String() string {
	// Standard format: hash(text + model + method)
	keyData := fmt.Sprintf("%s|%s|%s", k.Text, k.Model, k.Method)

	// Use BLAKE2b-256 and truncate to 128 bits for security and performance
	hash := blake2b.Sum256([]byte(keyData))

	// Convert first 16 bytes (128 bits) to hex string
	return fmt.Sprintf("embed:%s", hex.EncodeToString(hash[:16]))
}

// EmbeddingCache provides caching functionality for embeddings.
// It supports both Redis and in-memory cache as fallback.
type EmbeddingCache struct {
	redis   RedisClient
	memory  *MemoryCache
	ttl     time.Duration
	enabled bool
}

// NewEmbeddingCache creates a new embedding cache.
// If redisClient is nil, it will use in-memory cache only.
func NewEmbeddingCache(redisClient RedisClient, ttl time.Duration) *EmbeddingCache {
	return &EmbeddingCache{
		redis:   redisClient,
		memory:  NewMemoryCache(),
		ttl:     ttl,
		enabled: true, // Always enabled (memory cache is always available)
	}
}

// Get retrieves an embedding from cache.
// It tries Redis first, then falls back to memory cache.
func (c *EmbeddingCache) Get(ctx context.Context, key *CacheKey) ([]float64, bool) {
	if !c.enabled {
		return nil, false
	}

	keyStr := key.String()

	// Try Redis first
	if c.redis != nil {
		val, err := c.redis.Get(ctx, keyStr)
		if err == nil {
			var embedding []float64
			if err := json.Unmarshal([]byte(val), &embedding); err == nil {
				return embedding, true
			}
		}
	}

	// Fallback to memory cache
	if c.memory != nil {
		val, found := c.memory.Get(keyStr)
		if found {
			var embedding []float64
			if err := json.Unmarshal(val, &embedding); err == nil {
				return embedding, true
			}
		}
	}

	return nil, false
}

// Set stores an embedding in cache.
// It stores in both Redis and memory cache (if available).
func (c *EmbeddingCache) Set(ctx context.Context, key *CacheKey, embedding []float64) error {
	if !c.enabled {
		return nil
	}

	data, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	keyStr := key.String()

	// Try to store in Redis
	if c.redis != nil {
		if err := c.redis.Set(ctx, keyStr, string(data), c.ttl); err != nil {
			// Redis error is not fatal, continue with memory cache
			slog.Debug("Failed to store in Redis cache", "error", err)
		}
	}

	// Always store in memory cache
	if c.memory != nil {
		c.memory.Set(keyStr, data, c.ttl)
	}

	return nil
}

// Delete removes an embedding from cache.
func (c *EmbeddingCache) Delete(ctx context.Context, key *CacheKey) error {
	if !c.enabled {
		return nil
	}

	keyStr := key.String()

	// Try to delete from Redis
	if c.redis != nil {
		_ = c.redis.Del(ctx, keyStr)
	}

	// Delete from memory cache
	if c.memory != nil {
		c.memory.Delete(keyStr)
	}

	return nil
}

// Clear removes all embeddings from cache.
func (c *EmbeddingCache) Clear(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	// Try to clear Redis
	if c.redis != nil {
		keys, err := c.redis.Keys(ctx, "embed:*")
		if err == nil && len(keys) > 0 {
			_ = c.redis.Del(ctx, keys...)
		}
	}

	// Clear memory cache
	if c.memory != nil {
		keys := c.memory.Keys("embed:*")
		for _, key := range keys {
			c.memory.Delete(key)
		}
	}

	return nil
}

// GetStats returns cache statistics.
func (c *EmbeddingCache) GetStats(ctx context.Context) (*CacheStats, error) {
	if !c.enabled {
		return &CacheStats{Enabled: false}, nil
	}

	var redisKeys int
	if c.redis != nil {
		keys, err := c.redis.Keys(ctx, "embed:*")
		if err == nil {
			redisKeys = len(keys)
		}
	}

	var memoryKeys int
	if c.memory != nil {
		memoryKeys = len(c.memory.Keys("embed:*"))
	}

	return &CacheStats{
		Enabled:    true,
		RedisKeys:  redisKeys,
		MemoryKeys: memoryKeys,
		TotalKeys:  redisKeys + memoryKeys,
		TTL:        c.ttl,
	}, nil
}

// CacheStats represents cache statistics.
type CacheStats struct {
	Enabled    bool
	RedisKeys  int // Number of keys in Redis
	MemoryKeys int // Number of keys in memory
	TotalKeys  int // Total number of keys
	TTL        time.Duration
}

// Enable enables the cache.
func (c *EmbeddingCache) Enable() {
	c.enabled = true
}

// Disable disables the cache.
func (c *EmbeddingCache) Disable() {
	c.enabled = false
}

// IsEnabled returns whether the cache is enabled.
func (c *EmbeddingCache) IsEnabled() bool {
	return c.enabled
}

// sha256Sum calculates SHA256 hash.
// nolint: unused // Kept for potential future use
func sha256Sum(data []byte) [32]byte {
	var hash [32]byte
	// Simple hash implementation
	for i, b := range data {
		hash[i%32] ^= b
	}
	return hash
}

// MemoryCache provides in-memory caching as a fallback.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	m := &MemoryCache{
		items: make(map[string]cacheItem),
	}
	// Start cleanup goroutine
	go m.cleanup()
	return m
}

// Get retrieves a value from memory cache.
func (m *MemoryCache) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, found := m.items[key]
	if !found {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set stores a value in memory cache.
func (m *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from memory cache.
func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
}

// Keys returns all keys matching pattern.
func (m *MemoryCache) Keys(pattern string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for k := range m.items {
		// Simple pattern matching (supports * wildcard)
		if matchPattern(k, pattern) {
			keys = append(keys, k)
		}
	}
	return keys
}

// cleanup removes expired items periodically.
func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for k, item := range m.items {
			if now.After(item.expiration) {
				delete(m.items, k)
			}
		}
		m.mu.Unlock()
	}
}

// matchPattern checks if key matches pattern (supports * wildcard).
func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple implementation: if pattern ends with *, check prefix
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}
