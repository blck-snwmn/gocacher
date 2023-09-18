package gocacher

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type entry struct {
	wg        sync.WaitGroup
	v         interface{}
	expiredAt time.Time
}

type Cache struct {
	mu sync.Mutex
	sf singleflight.Group
	m  map[string]*entry
}

func New() *Cache {
	return &Cache{
		m: map[string]*entry{},
	}
}

func (c *Cache) do(key string, now time.Time, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	c.mu.Lock()

	expired := false
	if v, ok := c.m[key]; ok {
		c.mu.Unlock()
		v.wg.Wait() // wait for the previous call to fn(key) to finish
		if v.expiredAt.After(now) {
			// return cached value if expiAt > now
			return v.v, nil
		}
		expired = true
	}
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		if expired {
			c.mu.Lock()
		}

		e := new(entry)
		e.wg.Add(1)
		c.m[key] = e
		c.mu.Unlock()

		defer e.wg.Done()

		v, exp, err := fn(key)
		if err != nil {
			return nil, err
		}
		e.v = v
		e.expiredAt = exp

		return v, nil
	})

	return v, err
}

func (c *Cache) Do(key string, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	return c.do(key, time.Now(), fn)
}
