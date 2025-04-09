package network

import (
	"sync"
)

type cache struct {
	pos     int
	mu      sync.RWMutex
	buckets []map[string]struct{}
}

func newCache() *cache {
	bucks := make([]map[string]struct{}, 0, cacheBucketsCount)
	for range cacheBucketsCount {
		bucks = append(bucks, make(map[string]struct{}, cacheBucketSize))
	}

	return &cache{
		buckets: bucks,
	}
}

func (c *cache) putIfAbsent(hash string) bool {
	c.mu.RLock()
	pos := c.pos
	for i := pos; i >= 0; i-- {
		_, ok := c.buckets[i][hash]
		if ok {
			c.mu.RUnlock()
			return false
		}
	}
	c.mu.RUnlock()

	c.put(hash)

	return true
}

func (c *cache) put(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pos := c.pos
	if len(c.buckets[pos]) < cacheBucketSize {
		c.buckets[pos][hash] = struct{}{}
		return
	}

	pos++
	if pos >= cacheBucketsCount {
		pos = 0
	}

	c.buckets[pos] = make(map[string]struct{}, cacheBucketSize)
	c.buckets[pos][hash] = struct{}{}
}
