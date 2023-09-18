package gocacher

import (
	"errors"
	"fmt"
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
}

func TestCacheDoUseCache(t *testing.T) {
	c := New()

	var calls int64
	ch := make(chan struct{})
	f := func(key string) (interface{}, time.Time, error) {
		atomic.AddInt64(&calls, 1)
		<-ch
		return "bar", time.Now().Add(time.Minute), nil
	}
	var sg sync.WaitGroup
	for i := 0; i < 10; i++ {
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

func TestCacheDoDontUseCache(t *testing.T) {
	c := New()

	loop := 100

	var calls int64
	ch := make(chan struct{})
	f := func(key string) (interface{}, time.Time, error) {
		atomic.AddInt64(&calls, 1)
		<-ch
		return "bar", time.Now().Add(time.Minute), nil
	}
	var sg sync.WaitGroup
	for i := 0; i < loop; i++ {
		sg.Add(1)
		go func(i int) {
			defer sg.Done()

			v, err := c.Do(fmt.Sprintf("key-%d", i), f)
			if err != nil {
				t.Errorf("Do error = %v", err)
			}
			if got, want := v.(string), "bar"; got != want {
				t.Errorf("Do = %v; want %v", got, want)
			}
		}(i)
	}

	close(ch)
	sg.Wait()

	if calls != int64(loop) {
		t.Errorf("Do count = %v; want %d", calls, loop)
	}
}

// キャッシュが切れた場合、キャッシュを使わないことを確認するテスト
func TestCacheDo_DontUseCache_expired(t *testing.T) {
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

	loop := 100

	var calls int64
	ch := make(chan struct{})

	// call Do
	f := func(key string) (interface{}, time.Time, error) {
		atomic.AddInt64(&calls, 1)
		<-ch
		return "bar", time.Now().Add(10 * time.Millisecond), nil
	}
	var sg sync.WaitGroup
	for i := 0; i < loop-1; i++ {
		sg.Add(1)
		go func() {
			defer sg.Done()

			v, err := c.Do("key", f)
			if err != nil {
				t.Errorf("Do error = %v", err)
				return
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
