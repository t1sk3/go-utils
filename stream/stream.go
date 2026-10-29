// Package stream provides Stream, a lazy, chainable pipeline over a sequence of values.
//
// Intermediate operations (Filter, Map, Limit, Sorted, ...) only describe work; nothing
// runs until a terminal operation (ToSlice, ForEach, Count, Reduce, ...) is called.
// Elements then flow through the whole pipeline one at a time, so no intermediate
// slices are allocated, and operations such as Limit, FindFirst and AnyMatch stop
// pulling from the source as soon as they have their answer. This also makes
// infinite streams (Iterate, Generate) safe to use with Limit or TakeWhile.
//
// Go methods cannot introduce new type parameters, so operations that change the
// element type or need a constraint on it are package-level functions:
//
//	names := stream.Map(stream.FromSlice(users).Filter(User.IsActive), User.Name).ToSlice()
//	total := stream.Sum(stream.Map(stream.FromSlice(orders), Order.Amount))
//
// Streams are values and every operation returns a new Stream, so a Stream can be
// shared and extended freely. A Stream built from a slice, range or generator can be
// consumed any number of times; one built with FromSeq is only as reusable as the
// iter.Seq it wraps. The zero value is an empty Stream.
package stream

import (
	"iter"
	"maps"
	"math"
	"slices"

	"github.com/t1sk3/go-utils/types/constraints"
	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Stream is a lazy sequence of elements of type T. See the package documentation.
type Stream[T any] struct {
	// seq produces the elements. When nil, the stream is exactly slice with no
	// pending operations, which lets terminal operations take fast paths.
	seq   iter.Seq[T]
	slice []T
	// size is the exact number of elements when known, or -1. It is only used to
	// pre-size buffers.
	size int
}

// Of returns a Stream of the given values.
func Of[T any](values ...T) Stream[T] {
	return FromSlice(values)
}

// FromSlice returns a Stream over the elements of s. The slice is not copied, so it
// must not be modified while the stream is in use.
func FromSlice[T any](s []T) Stream[T] {
	return Stream[T]{slice: s, size: len(s)}
}

// FromArray returns a Stream over the elements of s.
//
// Deprecated: Use FromSlice.
func FromArray[T any](s []T) Stream[T] {
	return FromSlice(s)
}

// FromSeq returns a Stream over an iterator, such as maps.Keys(m) or slices.Values(s).
func FromSeq[T any](seq iter.Seq[T]) Stream[T] {
	if seq == nil {
		return Stream[T]{}
	}
	return Stream[T]{seq: seq, size: -1}
}

// FromSeq2 returns a Stream of the key-value pairs of an iterator, such as
// maps.All(m) or slices.All(s). Use ToSeq2 to turn it back into an iter.Seq2.
func FromSeq2[K, V any](seq iter.Seq2[K, V]) Stream[pair.Pair[K, V]] {
	if seq == nil {
		return Stream[pair.Pair[K, V]]{}
	}
	return Stream[pair.Pair[K, V]]{size: -1, seq: func(yield func(pair.Pair[K, V]) bool) {
		for k, v := range seq {
			if !yield(pair.Of(k, v)) {
				return
			}
		}
	}}
}

// FromMap returns a Stream of the entries of m, in no particular order. The map is
// not copied, so it must not be modified while the stream is in use.
func FromMap[K comparable, V any](m map[K]V) Stream[pair.Pair[K, V]] {
	s := FromSeq2(maps.All(m))
	s.size = len(m)
	return s
}

// Empty returns a Stream with no elements. It is equivalent to the zero value.
func Empty[T any]() Stream[T] {
	return Stream[T]{}
}

// Range returns a Stream of the integers from start (inclusive) to end (exclusive).
func Range[T constraints.Integer](start, end T) Stream[T] {
	size := 0
	if end > start {
		size = -1
		if d := end - start; d > 0 && uint64(d) <= math.MaxInt {
			size = int(d)
		}
	}
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		for v := start; v < end; v++ {
			if !yield(v) {
				return
			}
		}
	}}
}

// Iterate returns an infinite Stream of seed, next(seed), next(next(seed)), ...
// Bound it with Limit or TakeWhile before calling a terminal operation that
// consumes every element.
func Iterate[T any](seed T, next func(T) T) Stream[T] {
	checkFunc(next == nil, "Iterate")
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for v := seed; yield(v); v = next(v) {
		}
	}}
}

// Generate returns an infinite Stream whose elements are produced by calling supplier.
// Bound it with Limit or TakeWhile before calling a terminal operation that
// consumes every element.
func Generate[T any](supplier func() T) Stream[T] {
	checkFunc(supplier == nil, "Generate")
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for yield(supplier()) {
		}
	}}
}

// Concat returns a Stream of the elements of each stream in turn.
func Concat[T any](streams ...Stream[T]) Stream[T] {
	size := 0
	for _, s := range streams {
		if s.size < 0 || size > math.MaxInt-s.size {
			size = -1
			break
		}
		size += s.size
	}
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		for _, s := range streams {
			for v := range s.All() {
				if !yield(v) {
					return
				}
			}
		}
	}}
}

// All returns the stream as an iter.Seq, for use with range-over-func or other
// iterator-based APIs:
//
//	for v := range s.All() { ... }
func (s Stream[T]) All() iter.Seq[T] {
	if s.seq != nil {
		return s.seq
	}
	return slices.Values(s.slice)
}

// Filter returns a Stream of the elements that satisfy predicate.
func (s Stream[T]) Filter(predicate func(T) bool) Stream[T] {
	checkFunc(predicate == nil, "Filter")
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for v := range src {
			if predicate(v) && !yield(v) {
				return
			}
		}
	}}
}

// Map returns a Stream of the results of applying mapper to each element. To map to
// a different element type, use the package-level Map function.
func (s Stream[T]) Map(mapper func(T) T) Stream[T] {
	return Map(s, mapper)
}

// Peek returns a Stream that calls action on each element as it flows through,
// which is mainly useful for debugging.
func (s Stream[T]) Peek(action func(T)) Stream[T] {
	checkFunc(action == nil, "Peek")
	src := s.All()
	return Stream[T]{size: s.size, seq: func(yield func(T) bool) {
		for v := range src {
			action(v)
			if !yield(v) {
				return
			}
		}
	}}
}

// Limit returns a Stream of at most the first n elements. It never pulls more than
// n elements from the source, so it can be used to bound an infinite stream.
func (s Stream[T]) Limit(n int) Stream[T] {
	if n < 0 {
		panic("stream: Limit requires a non-negative count")
	}
	if s.seq == nil {
		return FromSlice(s.slice[:min(n, len(s.slice))])
	}
	size := -1
	if s.size >= 0 {
		size = min(n, s.size)
	}
	src := s.seq
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		if n == 0 {
			return
		}
		i := 0
		for v := range src {
			if !yield(v) {
				return
			}
			if i++; i == n {
				return
			}
		}
	}}
}

// Skip returns a Stream without the first n elements.
func (s Stream[T]) Skip(n int) Stream[T] {
	if n < 0 {
		panic("stream: Skip requires a non-negative count")
	}
	if s.seq == nil {
		return FromSlice(s.slice[min(n, len(s.slice)):])
	}
	size := -1
	if s.size >= 0 {
		size = max(s.size-n, 0)
	}
	src := s.seq
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		i := 0
		for v := range src {
			if i < n {
				i++
				continue
			}
			if !yield(v) {
				return
			}
		}
	}}
}

// TakeWhile returns a Stream of the leading elements that satisfy predicate,
// stopping at the first element that does not.
func (s Stream[T]) TakeWhile(predicate func(T) bool) Stream[T] {
	checkFunc(predicate == nil, "TakeWhile")
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for v := range src {
			if !predicate(v) || !yield(v) {
				return
			}
		}
	}}
}

// DropWhile returns a Stream that skips the leading elements that satisfy predicate
// and yields everything from the first element that does not.
func (s Stream[T]) DropWhile(predicate func(T) bool) Stream[T] {
	checkFunc(predicate == nil, "DropWhile")
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		dropping := true
		for v := range src {
			if dropping && predicate(v) {
				continue
			}
			dropping = false
			if !yield(v) {
				return
			}
		}
	}}
}

// Sorted returns a Stream of the elements in the order defined by comparator, which
// returns a negative number when a < b, zero when a == b and a positive number when
// a > b (cmp.Compare and strings.Compare are suitable). The sort is stable. To sort
// ordered types in their natural order, use the package-level Sort function.
func (s Stream[T]) Sorted(comparator func(a, b T) int) Stream[T] {
	checkFunc(comparator == nil, "Sorted")
	return s.sortedWith(func(buf []T) { slices.SortStableFunc(buf, comparator) })
}

// sortedWith returns a Stream that buffers every element, sorts the buffer in place
// with sortFn and then yields it. Sorting happens at iteration time, so the stream
// stays lazy.
func (s Stream[T]) sortedWith(sortFn func([]T)) Stream[T] {
	return Stream[T]{size: s.size, seq: func(yield func(T) bool) {
		buf := s.ToSlice()
		sortFn(buf)
		for _, v := range buf {
			if !yield(v) {
				return
			}
		}
	}}
}

// ForEach calls action for every element.
func (s Stream[T]) ForEach(action func(T)) {
	checkFunc(action == nil, "ForEach")
	for v := range s.All() {
		action(v)
	}
}

// AllMatch reports whether every element satisfies predicate. It returns true for
// an empty stream and stops at the first element that does not match.
func (s Stream[T]) AllMatch(predicate func(T) bool) bool {
	checkFunc(predicate == nil, "AllMatch")
	for v := range s.All() {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// AnyMatch reports whether any element satisfies predicate. It stops at the first match.
func (s Stream[T]) AnyMatch(predicate func(T) bool) bool {
	checkFunc(predicate == nil, "AnyMatch")
	for v := range s.All() {
		if predicate(v) {
			return true
		}
	}
	return false
}

// NoneMatch reports whether no element satisfies predicate. It stops at the first match.
func (s Stream[T]) NoneMatch(predicate func(T) bool) bool {
	checkFunc(predicate == nil, "NoneMatch")
	return !s.AnyMatch(predicate)
}

// FindFirst returns the first element, or an empty Optional if the stream is empty.
func (s Stream[T]) FindFirst() optional.Optional[T] {
	if s.seq == nil {
		if len(s.slice) == 0 {
			return optional.Empty[T]()
		}
		return optional.Of(s.slice[0])
	}
	for v := range s.seq {
		return optional.Of(v)
	}
	return optional.Empty[T]()
}

// FindAny returns some element, or an empty Optional if the stream is empty.
// Streams are sequential, so this is the same as FindFirst.
func (s Stream[T]) FindAny() optional.Optional[T] {
	return s.FindFirst()
}

// Count returns the number of elements.
func (s Stream[T]) Count() int {
	if s.seq == nil {
		return len(s.slice)
	}
	n := 0
	for range s.seq {
		n++
	}
	return n
}

// Min returns the smallest element according to comparator, or an empty Optional if
// the stream is empty. If several elements are equally small, the first is returned.
func (s Stream[T]) Min(comparator func(a, b T) int) optional.Optional[T] {
	checkFunc(comparator == nil, "Min")
	return s.Reduce(func(acc, v T) T {
		if comparator(v, acc) < 0 {
			return v
		}
		return acc
	})
}

// Max returns the largest element according to comparator, or an empty Optional if
// the stream is empty. If several elements are equally large, the first is returned.
func (s Stream[T]) Max(comparator func(a, b T) int) optional.Optional[T] {
	checkFunc(comparator == nil, "Max")
	return s.Reduce(func(acc, v T) T {
		if comparator(v, acc) > 0 {
			return v
		}
		return acc
	})
}

// Reduce combines the elements from left to right with accumulator, or returns an
// empty Optional if the stream is empty. To reduce into a different type or start
// from an initial value, use the package-level Fold function.
func (s Stream[T]) Reduce(accumulator func(acc, v T) T) optional.Optional[T] {
	checkFunc(accumulator == nil, "Reduce")
	var result T
	first := true
	for v := range s.All() {
		if first {
			result, first = v, false
		} else {
			result = accumulator(result, v)
		}
	}
	if first {
		return optional.Empty[T]()
	}
	return optional.Of(result)
}

// Partition collects the elements that satisfy predicate into matched and the rest
// into rest, preserving their order, in a single pass. Neither result is nil.
func (s Stream[T]) Partition(predicate func(T) bool) (matched, rest []T) {
	checkFunc(predicate == nil, "Partition")
	matched, rest = []T{}, []T{}
	for v := range s.All() {
		if predicate(v) {
			matched = append(matched, v)
		} else {
			rest = append(rest, v)
		}
	}
	return matched, rest
}

// ToSlice collects the elements into a new slice. The result is never nil and never
// shares memory with the stream's source.
func (s Stream[T]) ToSlice() []T {
	if s.seq == nil {
		out := make([]T, len(s.slice))
		copy(out, s.slice)
		return out
	}
	out := make([]T, 0, max(s.size, 0))
	for v := range s.seq {
		out = append(out, v)
	}
	return out
}

func checkFunc(isNil bool, op string) {
	if isNil {
		panic("stream: " + op + " requires a non-nil function")
	}
}
