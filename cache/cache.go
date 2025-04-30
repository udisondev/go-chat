package cache

import (
	"go-chat/config"
	"sync"
)

type Cache struct {
	mu      sync.Mutex
	pos     int
	buckets []*bucket
}

type bucket struct {
	mu   sync.Mutex
	cmap map[string]struct{}
}

func New() *Cache {
	buckets := make([]*bucket, config.CacheBucketsCount)
	for i := range buckets {
		buckets[i] = &bucket{cmap: make(map[string]struct{}, config.CacheBucketSize)}
	}
	return &Cache{
		buckets: buckets,
	}
}

func (c *Cache) Put(s string) {
	buck := c.buckets[c.pos]
	buck.mu.Lock()
	if len(buck.cmap) < config.CacheBucketSize {
		buck.cmap[s] = struct{}{}
		buck.mu.Unlock()
		return
	}
	buck.mu.Unlock()
	c.mu.Lock()
	c.pos++
	if c.pos < config.CacheBucketsCount {
		buck := c.buckets[c.pos]
		buck.mu.Lock()
		buck.cmap[s] = struct{}{}
		buck.mu.Unlock()
		c.mu.Unlock()
		return
	}
	c.pos = 0
	c.mu.Unlock()
	buck = c.buckets[c.pos]
	buck.mu.Lock()
	buck.cmap[s] = struct{}{}
	buck.mu.Unlock()
	return
}

func (c *Cache) PutIfAbsent(s string) bool {
	for i := range c.buckets {
		b := c.buckets[len(c.buckets)-1-i]
		b.mu.Lock()
		_, ok := b.cmap[s]
		if ok {
			b.mu.Unlock()
			return true
		}
		b.mu.Unlock()
	}
	c.Put(s)
	return false
}
