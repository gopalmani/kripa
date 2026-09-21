// SPDX-License-Identifier: AGPL-3.0-or-later
// Package memo provides bounded LRU storage and duplicate-request coalescing.
package memo

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type entry struct {
	key     string
	data    []byte
	expires time.Time
}
type flight struct {
	done chan struct{}
	data []byte
	err  error
}
type Cache struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	lru      *list.List
	items    map[string]*list.Element
	running  map[string]*flight
}

func New(capacity int, ttl time.Duration) *Cache {
	return &Cache{capacity: capacity, ttl: ttl, lru: list.New(), items: map[string]*list.Element{}, running: map[string]*flight{}}
}

// Callers must treat returned bytes as immutable. Capacity counts completed
// values; concurrent flights are bounded by the HTTP server admission limit.
func (c *Cache) Do(ctx context.Context, key string, fn func() ([]byte, error)) ([]byte, bool, error) {
	c.mu.Lock()
	if el := c.items[key]; el != nil {
		e := el.Value.(entry)
		if time.Now().Before(e.expires) {
			c.lru.MoveToFront(el)
			c.mu.Unlock()
			return e.data, true, nil
		}
		c.lru.Remove(el)
		delete(c.items, key)
	}
	if f := c.running[key]; f != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-f.done:
			return f.data, true, f.err
		}
	}
	f := &flight{done: make(chan struct{})}
	c.running[key] = f
	c.mu.Unlock()
	// Always release waiters, including if a native/provider implementation panics.
	defer func() {
		if v := recover(); v != nil {
			c.mu.Lock()
			f.err = context.Canceled
			delete(c.running, key)
			close(f.done)
			c.mu.Unlock()
			panic(v)
		}
	}()
	data, err := fn()
	c.mu.Lock()
	defer c.mu.Unlock()
	f.data, f.err = data, err
	if err == nil && c.capacity > 0 {
		el := c.lru.PushFront(entry{key, data, time.Now().Add(c.ttl)})
		c.items[key] = el
		for c.lru.Len() > c.capacity {
			last := c.lru.Back()
			delete(c.items, last.Value.(entry).key)
			c.lru.Remove(last)
		}
	}
	delete(c.running, key)
	close(f.done)
	return data, false, err
}
