// Package bimap provides BiMap, a one-to-one map that can be looked up from either
// side.
//
// Every key has one value and every value has one key, so a BiMap answers both "what
// is the value for this key" and "what is the key for this value" in constant time:
//
//	var codes bimap.BiMap[string, int]
//	codes.Put("NL", 31)
//	country := codes.GetKey(31) // "NL"
//
// The zero value is empty and ready to use. A BiMap must not be copied after first
// use, and it is not safe for concurrent use without external locking.
package bimap

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

// BiMap is a one-to-one map between keys and values. See the package documentation.
type BiMap[K, V comparable] struct {
	forward map[K]V
	reverse map[V]K
}

// Put pairs key with value. To keep the mapping one-to-one, it first removes any pair
// that already uses key or value, so a Put can remove up to two pairs.
func (m *BiMap[K, V]) Put(key K, value V) {
	if m.forward == nil {
		m.forward, m.reverse = make(map[K]V), make(map[V]K)
	}
	if old, ok := m.forward[key]; ok {
		delete(m.reverse, old)
	}
	if old, ok := m.reverse[value]; ok {
		delete(m.forward, old)
	}
	m.forward[key] = value
	m.reverse[value] = key
}

// Get returns the value for key, or an empty Optional if key is not present.
func (m *BiMap[K, V]) Get(key K) optional.Optional[V] {
	return optional.FromMap(m.forward, key)
}

// GetKey returns the key for value, or an empty Optional if value is not present.
func (m *BiMap[K, V]) GetKey(value V) optional.Optional[K] {
	return optional.FromMap(m.reverse, value)
}

// Has reports whether key is present.
func (m *BiMap[K, V]) Has(key K) bool {
	_, ok := m.forward[key]
	return ok
}

// HasValue reports whether value is present.
func (m *BiMap[K, V]) HasValue(value V) bool {
	_, ok := m.reverse[value]
	return ok
}

// Remove removes the pair with the given key and returns its value, if it was present.
func (m *BiMap[K, V]) Remove(key K) optional.Optional[V] {
	value, ok := m.forward[key]
	if ok {
		delete(m.forward, key)
		delete(m.reverse, value)
	}
	return optional.OfOk(value, ok)
}

// RemoveValue removes the pair with the given value and returns its key, if it was
// present.
func (m *BiMap[K, V]) RemoveValue(value V) optional.Optional[K] {
	key, ok := m.reverse[value]
	if ok {
		delete(m.forward, key)
		delete(m.reverse, value)
	}
	return optional.OfOk(key, ok)
}

// Len returns the number of pairs in m.
func (m *BiMap[K, V]) Len() int {
	return len(m.forward)
}

// IsEmpty reports whether m has no pairs.
func (m *BiMap[K, V]) IsEmpty() bool {
	return len(m.forward) == 0
}

// Clear removes every pair from m.
func (m *BiMap[K, V]) Clear() {
	clear(m.forward)
	clear(m.reverse)
}

// Clone returns a copy of m that does not share storage with it.
func (m *BiMap[K, V]) Clone() *BiMap[K, V] {
	return &BiMap[K, V]{forward: maps.Clone(m.forward), reverse: maps.Clone(m.reverse)}
}

// Inverse returns a new BiMap with keys and values swapped. It does not share storage
// with m.
func (m *BiMap[K, V]) Inverse() *BiMap[V, K] {
	return &BiMap[V, K]{forward: maps.Clone(m.reverse), reverse: maps.Clone(m.forward)}
}

// All returns an iterator over the pairs, in no particular order. m must not be
// modified during iteration.
func (m *BiMap[K, V]) All() iter.Seq2[K, V] {
	return maps.All(m.forward)
}

// String formats the pairs in sorted order of their %v representation, so output is
// stable across runs, for example "BiMap[a:1 b:2]".
func (m *BiMap[K, V]) String() string {
	parts := make([]string, 0, len(m.forward))
	for k, v := range m.forward {
		parts = append(parts, fmt.Sprintf("%v:%v", k, v))
	}
	slices.Sort(parts)
	return "BiMap[" + strings.Join(parts, " ") + "]"
}
