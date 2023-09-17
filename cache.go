package gocacher

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	v         interface{}
	expiredAt time.Time
}

type Cache struct {
	mu sync.Mutex
	m  map[string]entry
}

func New() *Cache {
	return &Cache{
		m: map[string]entry{},
	}
}

func (c *Cache) do(ctx context.Context, key string, now time.Time, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.m[key]; ok {
		// return cached value if expiAt > now
		if v.expiredAt.After(now) {
			return v.v, nil
		}
	}

	v, exp, err := fn(key)
	if err != nil {
		return nil, err
	}
	c.m[key] = entry{
		v:         v,
		expiredAt: exp,
	}
	return v, nil
}

func (c *Cache) Do(ctx context.Context, key string, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	return c.do(ctx, key, time.Now(), fn)
}
