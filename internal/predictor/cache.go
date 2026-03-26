// Package predictor provides caching for task statistics.
package predictor

import (
	"sync"
	"time"

	"algogpu/internal/db"
)

// StatsCache caches task statistics to avoid frequent database queries.
type StatsCache struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	stats     *db.TaskStats
	timestamp time.Time
}

// NewStatsCache creates a new stats cache with specified TTL.
func NewStatsCache(ttl time.Duration) *StatsCache {
	cache := &StatsCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	// Start cleanup routine
	go cache.cleanupRoutine()

	return cache
}

// Get retrieves stats from cache.
func (c *StatsCache) Get(taskType string) (*db.TaskStats, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[taskType]
	if !ok {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.stats, true
}

// Set stores stats in cache.
func (c *StatsCache) Set(taskType string, stats *db.TaskStats) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[taskType] = &cacheEntry{
		stats:     stats,
		timestamp: time.Now(),
	}
}

// Delete removes a task type from cache.
func (c *StatsCache) Delete(taskType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, taskType)
}

// Clear removes all entries from cache.
func (c *StatsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

// cleanupRoutine periodically removes expired entries.
func (c *StatsCache) cleanupRoutine() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpired()
	}
}

// cleanupExpired removes expired entries from cache.
func (c *StatsCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.cache, key)
		}
	}
}

// Size returns the number of entries in cache.
func (c *StatsCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}