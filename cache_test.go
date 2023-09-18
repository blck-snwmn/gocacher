package gocacher

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheDo(t *testing.T) {
	c := New()
	v, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		return "bar", time.Now().Add(time.Second), nil
	})
	if err != nil {
		t.Errorf("Do error = %v", err)
	}
	if got, want := v.(string), "bar"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
}

func TestCacheDoErr(t *testing.T) {
	c := New()

	someErr := errors.New("Some error")
	_, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		return "", time.Time{}, someErr
	})
	if err != someErr {
		t.Errorf("Do error = %v; want someErr %v", err, someErr)
	}
	_, _ = c.Do("key", func(key string) (interface{}, time.Time, error) {
		return "", time.Time{}, nil
	})
}

func TestCacheDo_UseCache(t *testing.T) {
	c := New()

	// register cache
	_, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		return "bar", time.Now().Add(time.Minute), nil
	})
	if err != nil {
		t.Errorf("Do error = %v", err)
		return
	}

	called := false
	v, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		called = true
		return "barz", time.Now().Add(10 * time.Millisecond), nil
	})
	if err != nil {
		t.Errorf("Do error = %v", err)
		return
	}
	if got, want := v.(string), "bar"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
	if called {
		t.Errorf("Do = %v; want %v", called, false)
	}
}

func TestCacheDo_UseNoCache_expired(t *testing.T) {
	c := New()

	// register cache
	_, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		return "bar", time.Now().Add(10 * time.Millisecond), nil
	})
	if err != nil {
		t.Errorf("Do error = %v", err)
		return
	}

	// wait for cache expired
	time.Sleep(11 * time.Millisecond)

	called := false
	v, err := c.Do("key", func(key string) (interface{}, time.Time, error) {
		called = true
		return "barz", time.Now().Add(10 * time.Millisecond), nil
	})
	if err != nil {
		t.Errorf("Do error = %v", err)
		return
	}
	if got, want := v.(string), "barz"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
	if !called {
		t.Errorf("Do = %v; want %v", called, true)
	}
}

func TestCacheDo_UseCache_parallel(t *testing.T) {
	c := New()

	var calls int64
	ch := make(chan struct{})
	f := func(key string) (interface{}, time.Time, error) {
		atomic.AddInt64(&calls, 1)
		<-ch
		return "bar", time.Now().Add(time.Minute), nil
	}
	var sg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		sg.Add(1)
		go func() {
			defer sg.Done()

			v, err := c.Do("key", f)
			if err != nil {
				t.Errorf("Do error = %v", err)
			}
			if got, want := v.(string), "bar"; got != want {
				t.Errorf("Do = %v; want %v", got, want)
			}
		}()
	}

	close(ch)
	sg.Wait()

	if calls != 1 {
		t.Errorf("Do count = %v; want 1", calls)
	}
}
