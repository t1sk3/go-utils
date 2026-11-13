package linked

import (
	"iter"

	"github.com/t1sk3/go-utils/types/optional"
)

// LRU is a cache that holds a fixed number of entries and, when full, evicts the one
// that was used least recently. Get and Put count as use; Peek and Has do not.
//
//	cache := linked.NewLRU[string, Page](1000)
//	if page, ok := cache.Get(url).Value(); ok {
//		return page
//	}
//	cache.Put(url, load(url))
//
// Every operation takes constant time. An LRU must be created with NewLRU. It must not
// be copied after first use, and it is not safe for concurrent use without external
// locking.
type LRU[K comparable, V any] struct {
	m        Map[K, V] // the least recently used entry is first
	capacity int
	onEvict  func(K, V)
}

// NewLRU returns an empty LRU that holds up to capacity entries. It panics if capacity
// is not positive.
func NewLRU[K comparable, V any](capacity int) *LRU[K, V] {
	if capacity <= 0 {
		panic("linked: NewLRU requires a positive capacity")
	}
	return &LRU[K, V]{capacity: capacity}
}

// OnEvict sets a function that is called with each entry the cache evicts to make
// room, for example to close a file or return a buffer to a pool. It is not called for
// entries removed with Remove or Clear, or for a value replaced by Put. It returns c so
// that it can follow NewLRU.
func (c *LRU[K, V]) OnEvict(f func(key K, value V)) *LRU[K, V] {
	c.onEvict = f
	return c
}

// Get returns the value for key and marks it as the most recently used, or returns an
// empty Optional if key is not present.
func (c *LRU[K, V]) Get(key K) optional.Optional[V] {
	v := c.m.Get(key)
	if v.IsPresent() {
		c.m.MoveToBack(key)
	}
	return v
}

// Peek returns the value for key without marking it as used.
func (c *LRU[K, V]) Peek(key K) optional.Optional[V] {
	return c.m.Get(key)
}

// Has reports whether key is present, without marking it as used.
func (c *LRU[K, V]) Has(key K) bool {
	return c.m.Has(key)
}

// Put sets the value for key and marks it as the most recently used. If that takes the
// cache over its capacity, the least recently used entry is evicted.
func (c *LRU[K, V]) Put(key K, value V) {
	c.m.Put(key, value)
	c.m.MoveToBack(key)
	c.evict()
}

// Remove removes key and returns its value, if it was present.
func (c *LRU[K, V]) Remove(key K) optional.Optional[V] {
	return c.m.Remove(key)
}

// Len returns the number of entries in c.
func (c *LRU[K, V]) Len() int {
	return c.m.Len()
}

// Cap returns the number of entries c can hold.
func (c *LRU[K, V]) Cap() int {
	return c.capacity
}

// Resize changes the number of entries c can hold, evicting the least recently used
// ones if it now holds too many. It panics if capacity is not positive.
func (c *LRU[K, V]) Resize(capacity int) {
	if capacity <= 0 {
		panic("linked: Resize requires a positive capacity")
	}
	c.capacity = capacity
	c.evict()
}

// Clear removes every entry from c.
func (c *LRU[K, V]) Clear() {
	c.m.Clear()
}

// All returns an iterator over the entries from least to most recently used, without
// marking them as used. The current entry may be removed during iteration; any other
// change makes the iteration unspecified.
func (c *LRU[K, V]) All() iter.Seq2[K, V] {
	return c.m.All()
}

func (c *LRU[K, V]) evict() {
	for c.m.Len() > c.capacity {
		e := c.m.PopFirst().MustGet()
		if c.onEvict != nil {
			c.onEvict(e.First, e.Second)
		}
	}
}
