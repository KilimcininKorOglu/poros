package enrich

import (
	"sync"
	"time"
)

// cacheEntry represents a single cache entry with expiration.
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// Cache is a simple thread-safe LRU-like cache with TTL.
type Cache struct {
	data     map[string]cacheEntry
	maxSize  int
	ttl      time.Duration
	mu       sync.RWMutex
	accesses map[string]time.Time // Track access times for eviction
}

// NewCache creates a new cache with the specified size and TTL.
func NewCache(maxSize int, ttl time.Duration) *Cache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	return &Cache{
		data:     make(map[string]cacheEntry),
		maxSize:  maxSize,
		ttl:      ttl,
		accesses: make(map[string]time.Time),
	}
}

// Get retrieves a value from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		delete(c.accesses, key)
		c.mu.Unlock()
		return nil, false
	}

	// Update access time
	c.mu.Lock()
	c.accesses[key] = time.Now()
	c.mu.Unlock()

	return entry.value, true
}

// Set stores a value in the cache.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.data) >= c.maxSize {
		c.evictOldest()
	}

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.accesses[key] = time.Now()
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.data) >= c.maxSize {
		c.evictOldest()
	}

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.accesses[key] = time.Now()
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	delete(c.accesses, key)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
	c.accesses = make(map[string]time.Time)
}

// Size returns the current number of entries in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// evictOldest removes the least recently accessed entry.
// Must be called with lock held.
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	first := true
	for key, accessTime := range c.accesses {
		if first || accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = accessTime
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		delete(c.accesses, oldestKey)
	}
}

// Cleanup removes expired entries.
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.expiresAt) {
			delete(c.data, key)
			delete(c.accesses, key)
		}
	}
}
