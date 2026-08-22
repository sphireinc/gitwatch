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
	value, _, err := c.get(ctx, key, fetch)
	return value, err
}

// GetWithStale returns expired data when a refresh fails, with stale indicating
// that callers should render the provider as degraded rather than available.
func (c *Cache[T]) GetWithStale(ctx context.Context, key string, fetch func(context.Context) (T, error)) (value T, stale bool, err error) {
	return c.get(ctx, key, fetch)
}

func (c *Cache[T]) get(ctx context.Context, key string, fetch func(context.Context) (T, error)) (T, bool, error) {
	now := time.Now()
	c.mu.Lock()
	item, ok := c.items[key]
	if ok && now.Sub(item.At) < c.ttl {
		c.mu.Unlock()
		return item.Value, false, nil
	}
	c.mu.Unlock()
	value, err := fetch(ctx)
	if err != nil {
		if ok {
			return item.Value, true, err
		}
		var zero T
		return zero, false, err
	}
	c.mu.Lock()
	c.items[key] = cacheItem[T]{Value: value, At: now}
	c.mu.Unlock()
	return value, false, nil
}
