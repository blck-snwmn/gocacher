package gocacher

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	wg        sync.WaitGroup
	v         interface{}
	expiredAt time.Time
}

type Cache struct {
	mu sync.Mutex
	m  map[string]*entry
}

func New() *Cache {
	return &Cache{
		m: map[string]*entry{},
	}
}

func (c *Cache) do(ctx context.Context, key string, now time.Time, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	c.mu.Lock()

	if v, ok := c.m[key]; ok {
		if v.expiredAt.After(now) {
			// return cached value if expiAt > now
			c.mu.Unlock()
			v.wg.Wait()
			return v.v, nil
		}
	}

	e := new(entry)
	e.wg.Add(1)
	c.m[key] = e
	c.mu.Unlock()

	v, exp, err := fn(key)
	if err != nil {
		return nil, err
	}
	e.v = v
	e.expiredAt = exp

	e.wg.Done()

	return v, nil
}

func (c *Cache) Do(ctx context.Context, key string, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	return c.do(ctx, key, time.Now(), fn)
}
