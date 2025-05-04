package cache

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Cache(t *testing.T) {
	t.Run("put one value", func(t *testing.T) {
		cache := New(2, 10)
		cache.Put(rand.Text())
		assert.Equal(t, 0, cache.pos)
	})
	t.Run("put 11 values", func(t *testing.T) {
		cache := New(2, 10)
		for range 11 {
			cache.Put(rand.Text())
		}
		assert.Equal(t, 1, cache.pos)
	})
	t.Run("fill in all buckts", func(t *testing.T) {
		cache := New(3, 10)
		for range 30 {
			cache.Put(rand.Text())
		}
		assert.Equal(t, 2, cache.pos)
		cache.Put(rand.Text())
		assert.Equal(t, 0, cache.pos)
		assert.Len(t, cache.buckets[0], 1)
		assert.Len(t, cache.buckets[1], 10)
		assert.Len(t, cache.buckets[2], 10)
	})
	t.Run("put existing", func(t *testing.T) {
		val := rand.Text()
		cache := New(1, 10)
		cache.Put(val)
		assert.Len(t, cache.buckets[0], 1)
		assert.False(t, cache.PutIfAbsent(val))
		assert.Len(t, cache.buckets[0], 1)
		assert.True(t, cache.PutIfAbsent(rand.Text()))
		assert.Len(t, cache.buckets[0], 2)
	})
}
