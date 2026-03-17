package cache

import (
	"context"
	"sync"
	"time"
)

// Cache — простой кэш с TTL. Для multi-instance заменить на Redis.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string, ttl time.Duration)
}

// MemoryCache — in-memory реализация с TTL.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	value     string
	expiresAt time.Time
}

func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{items: make(map[string]cacheItem)}
	go c.cleanupLoop()
	return c
}

func (c *MemoryCache) Get(ctx context.Context, key string) (string, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		return "", false
	}
	return item.value, true
}

func (c *MemoryCache) Set(ctx context.Context, key string, value string, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = cacheItem{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
