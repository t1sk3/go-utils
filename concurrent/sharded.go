package concurrent

import (
	"hash/maphash"
	"iter"
	"runtime"

	"github.com/t1sk3/go-utils/types/optional"
)

// ShardedMap is a Map split into independently locked parts, so that goroutines working
// on different keys rarely wait for each other. It has the same atomic operations per
// key as Map, and scales with the number of cores where Map's single lock does not.
//
// The price is that operations over the whole map are not atomic: Len, Clear, Snapshot,
// All and RemoveIf visit the shards one after another, so they can see some shards
// before and others after a concurrent change. Use Map when a consistent view of
// everything matters more than throughput.
//
// A ShardedMap must be created with NewShardedMap and must not be copied after first
// use. Functions passed to it run while one shard is locked and must not call methods
// of the same ShardedMap.
type ShardedMap[K comparable, V any] struct {
	seed   maphash.Seed
	shards []Map[K, V]
}

// NewShardedMap returns an empty ShardedMap with the given number of shards, rounded up
// to a power of two. Zero or less picks a number suited to the machine, several times
// its core count.
func NewShardedMap[K comparable, V any](shards int) *ShardedMap[K, V] {
	if shards <= 0 {
		shards = runtime.GOMAXPROCS(0) * 4
	}
	n := 1
	for n < shards {
		n <<= 1
	}
	return &ShardedMap[K, V]{seed: maphash.MakeSeed(), shards: make([]Map[K, V], n)}
}

// shard returns the part of the map that holds key.
func (m *ShardedMap[K, V]) shard(key K) *Map[K, V] {
	return &m.shards[maphash.Comparable(m.seed, key)&uint64(len(m.shards)-1)]
}

// Get returns the value for key, or an empty Optional if key is not present.
func (m *ShardedMap[K, V]) Get(key K) optional.Optional[V] { return m.shard(key).Get(key) }

// GetOrDefault returns the value for key, or fallback if key is not present.
func (m *ShardedMap[K, V]) GetOrDefault(key K, fallback V) V {
	return m.shard(key).GetOrDefault(key, fallback)
}

// Has reports whether key is present.
func (m *ShardedMap[K, V]) Has(key K) bool { return m.shard(key).Has(key) }

// Put sets the value for key and returns the previous value, if there was one.
func (m *ShardedMap[K, V]) Put(key K, value V) optional.Optional[V] {
	return m.shard(key).Put(key, value)
}

// PutIfAbsent sets the value for key if it is not present. It returns the value now in
// the map and whether it stored value.
func (m *ShardedMap[K, V]) PutIfAbsent(key K, value V) (actual V, stored bool) {
	return m.shard(key).PutIfAbsent(key, value)
}

// Replace sets the value for key only if key is present, and returns the previous value.
func (m *ShardedMap[K, V]) Replace(key K, value V) optional.Optional[V] {
	return m.shard(key).Replace(key, value)
}

// Remove removes key and returns its value, if it was present.
func (m *ShardedMap[K, V]) Remove(key K) optional.Optional[V] { return m.shard(key).Remove(key) }

// ComputeIfAbsent returns the value for key, storing the result of f first if key is
// not present. f runs at most once per missing key.
func (m *ShardedMap[K, V]) ComputeIfAbsent(key K, f func(K) V) V {
	return m.shard(key).ComputeIfAbsent(key, f)
}

// ComputeIfPresent replaces the value for key atomically if key is present, as
// Map.ComputeIfPresent does.
func (m *ShardedMap[K, V]) ComputeIfPresent(key K, f func(current V) (V, bool)) optional.Optional[V] {
	return m.shard(key).ComputeIfPresent(key, f)
}

// Compute replaces the entry for key atomically, as Map.Compute does.
func (m *ShardedMap[K, V]) Compute(key K, f func(current V, present bool) (V, bool)) optional.Optional[V] {
	return m.shard(key).Compute(key, f)
}

// Merge stores value if key is not present, and merge(existing, value) otherwise. It
// returns what is now stored.
func (m *ShardedMap[K, V]) Merge(key K, value V, merge func(existing, next V) V) V {
	return m.shard(key).Merge(key, value, merge)
}

// Len returns the number of entries. It is exact only if no other goroutine is changing
// the map.
func (m *ShardedMap[K, V]) Len() int {
	n := 0
	for i := range m.shards {
		n += m.shards[i].Len()
	}
	return n
}

// Clear removes every entry, one shard at a time.
func (m *ShardedMap[K, V]) Clear() {
	for i := range m.shards {
		m.shards[i].Clear()
	}
}

// RemoveIf removes the entries that satisfy predicate, one shard at a time, and returns
// how many it removed.
func (m *ShardedMap[K, V]) RemoveIf(predicate func(key K, value V) bool) int {
	n := 0
	for i := range m.shards {
		n += m.shards[i].RemoveIf(predicate)
	}
	return n
}

// Snapshot returns a copy of the entries as a plain map that the caller owns. Each
// shard is copied at a slightly different moment.
func (m *ShardedMap[K, V]) Snapshot() map[K]V {
	out := make(map[K]V)
	for k, v := range m.All() {
		out[k] = v
	}
	return out
}

// All returns an iterator over the entries, in no particular order. It copies one shard
// at a time and holds no lock while the loop body runs, so the body may call any method
// of m.
func (m *ShardedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := range m.shards {
			for k, v := range m.shards[i].Snapshot() {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}
