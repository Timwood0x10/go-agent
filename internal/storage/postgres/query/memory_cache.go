package query

import (
	"context"
	"sync"
	"time"
)

// MemoryQueryCache provides in-memory caching for query results.
type MemoryQueryCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	stopOnce sync.Once
}

type cacheItem struct {
	results   []*SearchResult
	expiresAt time.Time
}

// NewMemoryQueryCache creates a new in-memory query cache.
// The cleanup goroutine runs periodically to remove expired items.
func NewMemoryQueryCache() *MemoryQueryCache {
	ctx, cancel := context.WithCancel(context.Background())
	m := &MemoryQueryCache{
		items:  make(map[string]*cacheItem),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine with proper lifecycle management
	m.wg.Add(1)
	go m.cleanup()

	return m
}

// Close stops the cleanup goroutine and cleans up resources.
// This should be called when the cache is no longer needed to prevent goroutine leaks.
func (m *MemoryQueryCache) Close() {
	m.stopOnce.Do(func() {
		m.cancel()
		m.wg.Wait()

		// Clear all items
		m.mu.Lock()
		m.items = make(map[string]*cacheItem)
		m.mu.Unlock()
	})
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
// This goroutine runs until the context is cancelled or Close is called.
func (m *MemoryQueryCache) cleanup() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			// Context cancelled, stop cleanup
			return

		case <-ticker.C:
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
}

// Len returns the number of items in cache.
func (m *MemoryQueryCache) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.items)
}
