package provider

import (
	"context"
	"sync"
	"time"
)

type Cache[T any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]cacheItem[T]
}

type cacheItem[T any] struct {
	Value T
	At    time.Time
}

func NewCache[T any](ttl time.Duration) *Cache[T] {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &Cache[T]{ttl: ttl, items: make(map[string]cacheItem[T])}
}

func (c *Cache[T]) Get(ctx context.Context, key string, fetch func(context.Context) (T, error)) (T, error) {
	now := time.Now()
	c.mu.Lock()
	if item, ok := c.items[key]; ok && now.Sub(item.At) < c.ttl {
		c.mu.Unlock()
		return item.Value, nil
	}
	c.mu.Unlock()
	value, err := fetch(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	c.mu.Lock()
	c.items[key] = cacheItem[T]{Value: value, At: now}
	c.mu.Unlock()
	return value, nil
}
