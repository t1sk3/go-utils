// Package concurrent provides Map, a typed map that is safe for concurrent use.
//
// It is the counterpart of Java's ConcurrentHashMap in what it offers: compound updates
// such as ComputeIfAbsent and Merge that happen atomically, which sync.Map cannot do
// while running a function. It differs in how it works: a single read-write lock guards
// the whole map, so writers do not scale across many cores. For read-mostly maps with
// stable keys and no compound updates, prefer sync.Map.
package concurrent

import (
	"iter"
	"maps"
	"sync"

	"github.com/t1sk3/go-utils/types/optional"
)

// Map is a map that is safe for concurrent use. The zero value is empty and ready to
// use. A Map must not be copied after first use.
//
// Functions passed to ComputeIfAbsent, Compute and Merge run while the Map is locked.
// They must be short and must not call methods of the same Map, which would deadlock.
type Map[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// Get returns the value for key, or an empty Optional if key is not present.
func (m *Map[K, V]) Get(key K) optional.Optional[V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.m[key]
	return optional.OfOk(v, ok)
}

// Has reports whether key is present.
func (m *Map[K, V]) Has(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.m[key]
	return ok
}

// Put sets the value for key and returns the previous value, if there was one.
func (m *Map[K, V]) Put(key K, value V) optional.Optional[V] {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.m[key]
	m.set(key, value)
	return optional.OfOk(old, ok)
}

// PutIfAbsent sets the value for key if it is not present. It returns the value now in
// the Map and whether it stored value.
func (m *Map[K, V]) PutIfAbsent(key K, value V) (actual V, stored bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.m[key]; ok {
		return old, false
	}
	m.set(key, value)
	return value, true
}

// Remove removes key and returns its value, if it was present.
func (m *Map[K, V]) Remove(key K) optional.Optional[V] {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.m[key]
	delete(m.m, key)
	return optional.OfOk(old, ok)
}

// ComputeIfAbsent returns the value for key. If key is not present, it stores and
// returns the result of f, which runs at most once per missing key even when several
// goroutines ask at the same time. See Map for what f may do.
func (m *Map[K, V]) ComputeIfAbsent(key K, f func(K) V) V {
	// The common case needs only the read lock.
	m.mu.RLock()
	v, ok := m.m[key]
	m.mu.RUnlock()
	if ok {
		return v
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.m[key]; ok {
		return v
	}
	v = f(key)
	m.set(key, v)
	return v
}

// Compute replaces the entry for key atomically. f receives the current value, or the
// zero value and false if key is not present. It returns the new value, and false to
// remove the entry instead. Compute returns what is now stored. See Map for what f may do.
func (m *Map[K, V]) Compute(key K, f func(current V, present bool) (V, bool)) optional.Optional[V] {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, present := m.m[key]
	v, keep := f(current, present)
	if !keep {
		delete(m.m, key)
		return optional.Empty[V]()
	}
	m.set(key, v)
	return optional.Of(v)
}

// Merge stores value if key is not present. Otherwise it stores merge(existing, value).
// It returns what is now stored. See Map for what merge may do.
func (m *Map[K, V]) Merge(key K, value V, merge func(existing, next V) V) V {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.m[key]; ok {
		value = merge(existing, value)
	}
	m.set(key, value)
	return value
}

// Len returns the number of entries in m.
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// Clear removes every entry from m.
func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.m)
}

// Snapshot returns a copy of the entries as a plain map that the caller owns.
func (m *Map[K, V]) Snapshot() map[K]V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.m == nil {
		return map[K]V{}
	}
	return maps.Clone(m.m)
}

// All returns an iterator over a snapshot of the entries, in no particular order. The
// Map is not locked during the loop, so its body may call any method of m, and changes
// made after the snapshot was taken are not seen. Each iteration copies the entries.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m.Snapshot() {
			if !yield(k, v) {
				return
			}
		}
	}
}

// set stores an entry. The write lock must be held.
func (m *Map[K, V]) set(key K, value V) {
	if m.m == nil {
		m.m = make(map[K]V)
	}
	m.m[key] = value
}
