package cache

import (
	"sync"
	"time"
)

// Entry represents a cached item with expiration
type Entry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// TTLCache is a generic thread-safe cache with time-to-live expiration
type TTLCache[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]Entry[V]
	ttl     time.Duration
	maxSize int
}

// New creates a new TTL cache with the specified time-to-live and optional max size
// If maxSize is 0, the cache has no size limit
func New[K comparable, V any](ttl time.Duration, maxSize int) *TTLCache[K, V] {
	return &TTLCache[K, V]{
		items:   make(map[K]Entry[V]),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache if it exists and hasn't expired
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		var zero V
		return zero, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with TTL expiration
func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check size limit and evict if necessary
	if c.maxSize > 0 && len(c.items) >= c.maxSize {
		if _, exists := c.items[key]; !exists {
			// Only evict if we're adding a new key
			c.evictOldest()
		}
	}

	c.items[key] = Entry[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a key from the cache
func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *TTLCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]Entry[V])
}

// Size returns the current number of items in the cache
func (c *TTLCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictOldest removes the oldest entry (by expiration time) from the cache
// Must be called with lock held
func (c *TTLCache[K, V]) evictOldest() {
	if len(c.items) == 0 {
		return
	}

	var oldestKey K
	var oldestTime time.Time
	first := true

	for key, entry := range c.items {
		if first || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
			first = false
		}
	}

	delete(c.items, oldestKey)
}
