package dns

import (
	"sync"
	"time"
)

// CacheEntry holds a cached DNS response with expiry.
type CacheEntry struct {
	Records   []ResourceRecord
	ExpiresAt time.Time
}

// Cache provides thread-safe DNS response caching with TTL expiry.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

// cacheKey builds a cache key from name and type.
func cacheKey(name string, qtype uint16) string {
	return name + ":" + TypeToString(qtype)
}

// Get retrieves cached records for a name and type, returning nil if not found or expired.
func (c *Cache) Get(name string, qtype uint16) []ResourceRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(name, qtype)
	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	// Return a copy to prevent mutation.
	result := make([]ResourceRecord, len(entry.Records))
	copy(result, entry.Records)
	return result
}

// Put stores records in the cache. TTL is derived from the minimum TTL of the records.
func (c *Cache) Put(name string, qtype uint16, records []ResourceRecord) {
	if len(records) == 0 {
		return
	}

	// Find minimum TTL.
	minTTL := records[0].TTL
	for _, rr := range records[1:] {
		if rr.TTL < minTTL {
			minTTL = rr.TTL
		}
	}

	// Enforce a minimum cache time of 30 seconds.
	if minTTL < 30 {
		minTTL = 30
	}

	entry := &CacheEntry{
		Records:   make([]ResourceRecord, len(records)),
		ExpiresAt: time.Now().Add(time.Duration(minTTL) * time.Second),
	}
	copy(entry.Records, records)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cacheKey(name, qtype)] = entry
}

// Purge removes all expired entries from the cache.
func (c *Cache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}
