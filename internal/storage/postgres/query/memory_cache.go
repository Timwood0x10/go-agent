package query

import (
	"sync"
	"time"
)

// MemoryQueryCache provides in-memory caching for query results.
type MemoryQueryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	results   []*SearchResult
	expiresAt time.Time
}

// NewMemoryQueryCache creates a new in-memory query cache.
func NewMemoryQueryCache() *MemoryQueryCache {
	m := &MemoryQueryCache{
		items: make(map[string]*cacheItem),
	}
	// Start cleanup goroutine
	go m.cleanup()
	return m
}

// Get retrieves search results from cache.
func (m *MemoryQueryCache) Get(key string) ([]*SearchResult, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, found := m.items[key]
	if !found {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.results, true
}

// Set stores search results in cache.
func (m *MemoryQueryCache) Set(key string, results []*SearchResult, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = &cacheItem{
		results:   results,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes search results from cache.
func (m *MemoryQueryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
}

// Clear removes all items from cache.
func (m *MemoryQueryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]*cacheItem)
}

// cleanup removes expired items periodically.
func (m *MemoryQueryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for k, item := range m.items {
			if now.After(item.expiresAt) {
				delete(m.items, k)
			}
		}
		m.mu.Unlock()
	}
}

// Len returns the number of items in cache.
func (m *MemoryQueryCache) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.items)
}
