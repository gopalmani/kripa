package memo

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoalescingAndEviction(t *testing.T) {
	c := New(1, time.Hour)
	var calls atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			v, _, err := c.Do(context.Background(), "a", func() ([]byte, error) { calls.Add(1); time.Sleep(time.Millisecond); return []byte("a"), nil })
			if err != nil || string(v) != "a" {
				t.Error("invalid result")
			}
		}()
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("duplicate compute")
	}
	_, _, _ = c.Do(context.Background(), "b", func() ([]byte, error) { return []byte("b"), nil })
	_, hit, _ := c.Do(context.Background(), "a", func() ([]byte, error) { return []byte("a"), nil })
	if hit {
		t.Fatal("LRU did not evict")
	}
}
func TestErrorsNotCached(t *testing.T) {
	c := New(2, time.Hour)
	for i := 0; i < 2; i++ {
		_, hit, _ := c.Do(context.Background(), "x", func() ([]byte, error) { return nil, errors.New("error") })
		if hit {
			t.Fatal("cached an error")
		}
	}
}
func TestExpiry(t *testing.T) {
	c := New(2, -time.Second)
	f := func() ([]byte, error) { return []byte("v"), nil }
	_, _, _ = c.Do(context.Background(), "x", f)
	_, hit, _ := c.Do(context.Background(), "x", f)
	if hit {
		t.Fatal("expired cache hit")
	}
}
