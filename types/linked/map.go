// Package linked provides Map and Set, hash-based collections that remember the order
// in which entries were added. They are the counterparts of Java's LinkedHashMap and
// LinkedHashSet.
//
// Lookups, insertions and removals take constant time, as with a built-in map, and
// iteration follows insertion order. Because MoveToBack and PopFirst are also constant
// time, a Map makes a compact LRU cache:
//
//	if v, ok := cache.Get(key).Value(); ok {
//		cache.MoveToBack(key)
//		return v
//	}
//	cache.Put(key, load(key))
//	if cache.Len() > limit {
//		cache.PopFirst()
//	}
//
// The zero values are empty and ready to use. A Map or Set must not be copied after
// first use, and neither is safe for concurrent use without external locking.
package linked

import (
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Map is a hash map that iterates in insertion order. See the package documentation.
type Map[K comparable, V any] struct {
	nodes       map[K]*node[K, V]
	first, last *node[K, V]
}

type node[K comparable, V any] struct {
	key        K
	value      V
	prev, next *node[K, V]
}

// NewMap returns an empty Map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{}
}

// Put sets the value for key and returns the previous value, if there was one. A new
// key goes to the back. An existing key keeps its position; call MoveToBack as well to
// move it.
func (m *Map[K, V]) Put(key K, value V) optional.Optional[V] {
	if n, ok := m.nodes[key]; ok {
		old := n.value
		n.value = value
		return optional.Of(old)
	}
	if m.nodes == nil {
		m.nodes = make(map[K]*node[K, V])
	}
	n := &node[K, V]{key: key, value: value}
	m.nodes[key] = n
	m.linkLast(n)
	return optional.Empty[V]()
}

// Get returns the value for key, or an empty Optional if key is not present.
func (m *Map[K, V]) Get(key K) optional.Optional[V] {
	if n, ok := m.nodes[key]; ok {
		return optional.Of(n.value)
	}
	return optional.Empty[V]()
}

// Has reports whether key is present.
func (m *Map[K, V]) Has(key K) bool {
	_, ok := m.nodes[key]
	return ok
}

// Remove removes key and returns its value, or returns an empty Optional if key was
// not present.
func (m *Map[K, V]) Remove(key K) optional.Optional[V] {
	n, ok := m.nodes[key]
	if !ok {
		return optional.Empty[V]()
	}
	delete(m.nodes, key)
	m.unlink(n)
	return optional.Of(n.value)
}

// MoveToBack makes key the last entry and reports whether it was present.
func (m *Map[K, V]) MoveToBack(key K) bool {
	n, ok := m.nodes[key]
	if !ok {
		return false
	}
	if n != m.last {
		m.unlink(n)
		m.linkLast(n)
	}
	return true
}

// MoveToFront makes key the first entry and reports whether it was present.
func (m *Map[K, V]) MoveToFront(key K) bool {
	n, ok := m.nodes[key]
	if !ok {
		return false
	}
	if n != m.first {
		m.unlink(n)
		n.next = m.first
		m.first.prev = n
		m.first = n
	}
	return true
}

// Len returns the number of entries in m.
func (m *Map[K, V]) Len() int {
	return len(m.nodes)
}

// IsEmpty reports whether m has no entries.
func (m *Map[K, V]) IsEmpty() bool {
	return len(m.nodes) == 0
}

// Clear removes every entry from m.
func (m *Map[K, V]) Clear() {
	clear(m.nodes)
	m.first, m.last = nil, nil
}

// Clone returns a copy of m in the same order that does not share storage with it.
// Keys and values are copied by assignment.
func (m *Map[K, V]) Clone() *Map[K, V] {
	c := &Map[K, V]{nodes: make(map[K]*node[K, V], len(m.nodes))}
	for k, v := range m.All() {
		c.Put(k, v)
	}
	return c
}

// First returns the oldest entry, or an empty Optional if m is empty.
func (m *Map[K, V]) First() optional.Optional[pair.Pair[K, V]] {
	return m.first.entry()
}

// Last returns the newest entry, or an empty Optional if m is empty.
func (m *Map[K, V]) Last() optional.Optional[pair.Pair[K, V]] {
	return m.last.entry()
}

// PopFirst removes and returns the oldest entry, or returns an empty Optional if m is
// empty.
func (m *Map[K, V]) PopFirst() optional.Optional[pair.Pair[K, V]] {
	e := m.first.entry()
	if m.first != nil {
		m.Remove(m.first.key)
	}
	return e
}

// PopLast removes and returns the newest entry, or returns an empty Optional if m is
// empty.
func (m *Map[K, V]) PopLast() optional.Optional[pair.Pair[K, V]] {
	e := m.last.entry()
	if m.last != nil {
		m.Remove(m.last.key)
	}
	return e
}

// All returns an iterator over the entries from oldest to newest. The current entry
// may be removed during iteration; any other change makes the iteration unspecified.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for n := m.first; n != nil; {
			next := n.next
			if !yield(n.key, n.value) {
				return
			}
			n = next
		}
	}
}

// Backward returns an iterator over the entries from newest to oldest. The current
// entry may be removed during iteration; any other change makes the iteration
// unspecified.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for n := m.last; n != nil; {
			prev := n.prev
			if !yield(n.key, n.value) {
				return
			}
			n = prev
		}
	}
}

// Keys returns an iterator over the keys from oldest to newest, under the same
// conditions as All.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range m.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over the values from oldest to newest, under the same
// conditions as All.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// String formats the entries from oldest to newest, for example "Map[b:2 a:1]".
func (m *Map[K, V]) String() string {
	var b strings.Builder
	b.WriteString("Map[")
	for n := m.first; n != nil; n = n.next {
		if n != m.first {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%v:%v", n.key, n.value)
	}
	b.WriteByte(']')
	return b.String()
}

func (m *Map[K, V]) linkLast(n *node[K, V]) {
	n.prev, n.next = m.last, nil
	if m.last != nil {
		m.last.next = n
	} else {
		m.first = n
	}
	m.last = n
}

// unlink removes n from the list, leaving its prev and next unset.
func (m *Map[K, V]) unlink(n *node[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		m.first = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		m.last = n.prev
	}
	n.prev, n.next = nil, nil
}

func (n *node[K, V]) entry() optional.Optional[pair.Pair[K, V]] {
	if n == nil {
		return optional.Empty[pair.Pair[K, V]]()
	}
	return optional.Of(pair.Of(n.key, n.value))
}
