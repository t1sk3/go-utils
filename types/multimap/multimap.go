// Package multimap provides Multimap, a map that holds any number of values per key.
//
// It replaces the map[K][]V bookkeeping of checking for a missing key, appending, and
// deleting a key when its last value goes:
//
//	var byTag multimap.Multimap[string, Post]
//	byTag.Put("go", post1, post2)
//	for _, p := range byTag.Get("go") { ... }
//
// Values keep the order they were added in, and a key may hold the same value more
// than once. The zero value is empty and ready to use. A Multimap must not be copied
// after first use, and it is not safe for concurrent use without external locking.
package multimap

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// Multimap maps each key to a list of values. See the package documentation.
type Multimap[K comparable, V any] struct {
	m map[K][]V
	n int
}

// Put adds values to the list for key, in order.
func (m *Multimap[K, V]) Put(key K, values ...V) {
	if len(values) == 0 {
		return
	}
	if m.m == nil {
		m.m = make(map[K][]V)
	}
	m.m[key] = append(m.m[key], values...)
	m.n += len(values)
}

// Get returns the values for key in the order they were added, or nil if key is not
// present. The slice is the Multimap's own storage: it must not be modified, and it is
// only valid until the next change to m. Appending to it is safe and makes a copy.
func (m *Multimap[K, V]) Get(key K) []V {
	values := m.m[key]
	return values[:len(values):len(values)]
}

// Has reports whether key has at least one value.
func (m *Multimap[K, V]) Has(key K) bool {
	_, ok := m.m[key]
	return ok
}

// Count returns the number of values for key.
func (m *Multimap[K, V]) Count(key K) int {
	return len(m.m[key])
}

// Remove removes key with all its values and returns them, or nil if key was not
// present. The caller owns the returned slice.
func (m *Multimap[K, V]) Remove(key K) []V {
	values := m.m[key]
	delete(m.m, key)
	m.n -= len(values)
	return values
}

// RemoveFunc removes the values for key that satisfy predicate and returns how many it
// removed. The key goes too when no value is left.
func (m *Multimap[K, V]) RemoveFunc(key K, predicate func(V) bool) int {
	values, ok := m.m[key]
	if !ok {
		return 0
	}
	kept := slices.DeleteFunc(values, predicate)
	removed := len(values) - len(kept)
	m.n -= removed
	if len(kept) == 0 {
		delete(m.m, key)
	} else {
		m.m[key] = kept
	}
	return removed
}

// Len returns the total number of values, counting every value of every key.
func (m *Multimap[K, V]) Len() int {
	return m.n
}

// KeyLen returns the number of distinct keys.
func (m *Multimap[K, V]) KeyLen() int {
	return len(m.m)
}

// IsEmpty reports whether m has no values.
func (m *Multimap[K, V]) IsEmpty() bool {
	return m.n == 0
}

// Clear removes every key and value from m.
func (m *Multimap[K, V]) Clear() {
	clear(m.m)
	m.n = 0
}

// Clone returns a copy of m that does not share storage with it. Keys and values are
// copied by assignment.
func (m *Multimap[K, V]) Clone() *Multimap[K, V] {
	c := &Multimap[K, V]{m: make(map[K][]V, len(m.m)), n: m.n}
	for k, values := range m.m {
		c.m[k] = slices.Clone(values)
	}
	return c
}

// Keys returns an iterator over the distinct keys, in no particular order. m must not
// be modified during iteration.
func (m *Multimap[K, V]) Keys() iter.Seq[K] {
	return maps.Keys(m.m)
}

// All returns an iterator over every key-value pair. Keys come in no particular order,
// and a key with several values is produced once for each, in the order they were
// added. m must not be modified during iteration.
func (m *Multimap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, values := range m.m {
			for _, v := range values {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// Groups returns an iterator over each key with all of its values, in no particular
// order. The slices are the Multimap's own storage, under the same conditions as Get.
// m must not be modified during iteration.
func (m *Multimap[K, V]) Groups() iter.Seq2[K, []V] {
	return func(yield func(K, []V) bool) {
		for k, values := range m.m {
			if !yield(k, values[:len(values):len(values)]) {
				return
			}
		}
	}
}

// String formats the keys in sorted order of their %v representation, so output is
// stable across runs, for example "Multimap[a:[1 2] b:[3]]".
func (m *Multimap[K, V]) String() string {
	parts := make([]string, 0, len(m.m))
	for k, values := range m.m {
		parts = append(parts, fmt.Sprintf("%v:%v", k, values))
	}
	slices.Sort(parts)
	return "Multimap[" + strings.Join(parts, " ") + "]"
}
