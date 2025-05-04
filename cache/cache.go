package cache

import (
	"sync"
)

type Cache struct {
	mu          sync.Mutex
	pos         int
	buckets     []map[string]struct{}
	count, size int
}

func New(bucketsCount, bucketSize int) *Cache {
	bucks := make([]map[string]struct{}, 0, bucketsCount)
	for range bucketsCount {
		bucks = append(bucks, make(map[string]struct{}, bucketSize))
	}
	return &Cache{
		buckets: bucks,
		count:   bucketsCount,
		size:    bucketSize,
	}
}

func (c *Cache) Put(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.put(s)
}

func (c *Cache) PutIfAbsent(s string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := c.pos; i >= 0; i-- {
		_, ok := c.buckets[i][s]
		if ok {
			return false
		}
	}

	c.put(s)
	return true
}

func (c *Cache) put(s string) {
	if len(c.buckets[c.pos]) < c.size {
		c.buckets[c.pos][s] = struct{}{}
		return
	}

	c.pos++

	if c.pos >= c.count {
		c.pos = 0
	}

	c.buckets[c.pos] = make(map[string]struct{}, c.size)

	c.buckets[c.pos][s] = struct{}{}
}
