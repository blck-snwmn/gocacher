package gocacher

import (
	"errors"
	"sync"
	"time"
)

var ErrExpired = errors.New("expired")

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

func (c *Cache) do(key string, now time.Time, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	c.mu.Lock()

	if v, ok := c.m[key]; ok {
		c.mu.Unlock()
		v.wg.Wait() // wait for the previous call to fn(key) to finish
		if v.expiredAt.After(now) {
			// return cached value if expiAt > now
			return v.v, nil
		}
		// remove the expired entry
		c.mu.Lock()
		if !v.expiredAt.After(now) {
			// remove the expired entry if expiAt <= now
			// because the entry may be updated by other goroutine
			delete(c.m, key)
		}
		c.mu.Unlock()
		return v.v, ErrExpired
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

func (c *Cache) Do(key string, fn func(key string) (interface{}, time.Time, error)) (interface{}, error) {
	v, err := c.do(key, time.Now(), fn)
	if err == ErrExpired {
		v, err = c.do(key, time.Now(), fn)
	}
	return v, err
}
