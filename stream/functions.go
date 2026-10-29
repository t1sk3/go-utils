package stream

import (
	"cmp"
	"iter"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/constraints"
	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
	"github.com/t1sk3/go-utils/types/set"
)

// This file holds operations that Go cannot express as methods on Stream, because
// they introduce a new type parameter or constrain T.

// Map returns a Stream of the results of applying mapper to each element of s.
func Map[T, R any](s Stream[T], mapper func(T) R) Stream[R] {
	checkFunc(mapper == nil, "Map")
	src := s.All()
	return Stream[R]{size: s.size, seq: func(yield func(R) bool) {
		for v := range src {
			if !yield(mapper(v)) {
				return
			}
		}
	}}
}

// FlatMap returns a Stream of the elements of each stream produced by applying
// mapper to the elements of s.
func FlatMap[T, R any](s Stream[T], mapper func(T) Stream[R]) Stream[R] {
	checkFunc(mapper == nil, "FlatMap")
	src := s.All()
	return Stream[R]{size: -1, seq: func(yield func(R) bool) {
		for v := range src {
			for r := range mapper(v).All() {
				if !yield(r) {
					return
				}
			}
		}
	}}
}

// Distinct returns a Stream of the elements of s with duplicates removed, keeping
// the first occurrence of each.
func Distinct[T comparable](s Stream[T]) Stream[T] {
	return DistinctBy(s, func(v T) T { return v })
}

// DistinctBy returns a Stream of the elements of s whose key is not shared by an
// earlier element. Use it for element types that are not comparable.
func DistinctBy[T any, K comparable](s Stream[T], key func(T) K) Stream[T] {
	checkFunc(key == nil, "DistinctBy")
	src := s.All()
	hint := max(s.size, 0)
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		seen := make(map[K]struct{}, hint)
		for v := range src {
			k := key(v)
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}}
}

// Fold combines the elements of s from left to right, starting from initial. Unlike
// Stream.Reduce, the result may have a different type than the elements:
//
//	totalLen := stream.Fold(words, 0, func(n int, w string) int { return n + len(w) })
func Fold[T, A any](s Stream[T], initial A, accumulator func(acc A, v T) A) A {
	checkFunc(accumulator == nil, "Fold")
	acc := initial
	for v := range s.All() {
		acc = accumulator(acc, v)
	}
	return acc
}

// Sum returns the sum of the elements of s, or zero for an empty stream.
func Sum[T constraints.Number](s Stream[T]) T {
	var total T
	for v := range s.All() {
		total += v
	}
	return total
}

// Sort returns a Stream of the elements of s in ascending natural order. Use
// Stream.Sorted for a custom ordering.
func Sort[T cmp.Ordered](s Stream[T]) Stream[T] {
	return s.sortedWith(slices.Sort[[]T])
}

// Min returns the smallest element of s, or an empty Optional if s is empty.
func Min[T cmp.Ordered](s Stream[T]) optional.Optional[T] {
	return s.Min(cmp.Compare[T])
}

// Max returns the largest element of s, or an empty Optional if s is empty.
func Max[T cmp.Ordered](s Stream[T]) optional.Optional[T] {
	return s.Max(cmp.Compare[T])
}

// GroupBy collects the elements of s into a map from key to the elements with that
// key, preserving their order.
func GroupBy[T any, K comparable](s Stream[T], key func(T) K) map[K][]T {
	checkFunc(key == nil, "GroupBy")
	groups := make(map[K][]T)
	for v := range s.All() {
		k := key(v)
		groups[k] = append(groups[k], v)
	}
	return groups
}

// ToMap collects the elements of s into a map. If several elements produce the same
// key, the last one wins.
func ToMap[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V) map[K]V {
	checkFunc(key == nil || value == nil, "ToMap")
	m := make(map[K]V, max(s.size, 0))
	for v := range s.All() {
		m[key(v)] = value(v)
	}
	return m
}

// ToSet collects the elements of s into a set.Set.
func ToSet[T comparable](s Stream[T]) set.Set[T] {
	return set.Collect(s.All())
}

// ToSeq2 returns a stream of pairs as an iter.Seq2, for use with two-value range
// loops or functions such as maps.Collect:
//
//	byID := maps.Collect(stream.ToSeq2(stream.Zip(ids, users)))
func ToSeq2[K, V any](s Stream[pair.Pair[K, V]]) iter.Seq2[K, V] {
	src := s.All()
	return func(yield func(K, V) bool) {
		for p := range src {
			if !yield(p.First, p.Second) {
				return
			}
		}
	}
}

// Zip returns a Stream that pairs the elements of a and b by position. It ends when
// either stream ends, so zipping with an infinite stream is fine.
//
// When b comes straight from FromSlice or Of, Zip indexes it directly; otherwise it
// has to step b with iter.Pull, which is roughly ten times slower per element. If
// only one side is a plain slice, pass it as b.
func Zip[A, B any](a Stream[A], b Stream[B]) Stream[pair.Pair[A, B]] {
	size := -1
	if a.size >= 0 && b.size >= 0 {
		size = min(a.size, b.size)
	}
	srcA := a.All()
	if b.seq == nil {
		// b is a plain slice: index it instead of starting a coroutine with iter.Pull.
		bs := b.slice
		return Stream[pair.Pair[A, B]]{size: size, seq: func(yield func(pair.Pair[A, B]) bool) {
			if len(bs) == 0 {
				return
			}
			i := 0
			for va := range srcA {
				if !yield(pair.Of(va, bs[i])) {
					return
				}
				if i++; i == len(bs) {
					return
				}
			}
		}}
	}
	srcB := b.seq
	return Stream[pair.Pair[A, B]]{size: size, seq: func(yield func(pair.Pair[A, B]) bool) {
		nextB, stop := iter.Pull(srcB)
		defer stop()
		for va := range srcA {
			vb, ok := nextB()
			if !ok || !yield(pair.Of(va, vb)) {
				return
			}
		}
	}}
}

// Enumerate returns a Stream that pairs each element of s with its index, starting at 0.
func Enumerate[T any](s Stream[T]) Stream[pair.Pair[int, T]] {
	src := s.All()
	return Stream[pair.Pair[int, T]]{size: s.size, seq: func(yield func(pair.Pair[int, T]) bool) {
		i := 0
		for v := range src {
			if !yield(pair.Of(i, v)) {
				return
			}
			i++
		}
	}}
}

// Chunk returns a Stream of consecutive, non-overlapping slices of n elements of s.
// The last chunk is shorter if the elements do not divide evenly. Each chunk is a
// new slice that the caller may keep or modify. Chunk panics if n is not positive.
//
//	stream.Chunk(stream.Range(0, 5), 2) // [0 1] [2 3] [4]
func Chunk[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		panic("stream: Chunk requires a positive size")
	}
	size := -1
	if s.size >= 0 {
		size = s.size / n
		if s.size%n != 0 {
			size++
		}
	}
	src := s.All()
	capacity := capFor(n, s.size)
	return Stream[[]T]{size: size, seq: func(yield func([]T) bool) {
		var buf []T
		for v := range src {
			if buf == nil {
				buf = make([]T, 0, capacity)
			}
			buf = append(buf, v)
			if len(buf) == n {
				if !yield(buf) {
					return
				}
				buf = nil
			}
		}
		if len(buf) > 0 {
			yield(buf)
		}
	}}
}

// Window returns a Stream of every run of n consecutive elements of s, advancing one
// element at a time. A stream with fewer than n elements produces no windows. Each
// window is a new slice that the caller may keep or modify. Window panics if n is
// not positive.
//
//	stream.Window(stream.Range(0, 4), 2) // [0 1] [1 2] [2 3]
func Window[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		panic("stream: Window requires a positive size")
	}
	size := -1
	if s.size >= 0 {
		size = max(s.size-n+1, 0)
	}
	src := s.All()
	capacity := capFor(n, s.size)
	return Stream[[]T]{size: size, seq: func(yield func([]T) bool) {
		buf := make([]T, 0, capacity)
		for v := range src {
			if len(buf) == n {
				copy(buf, buf[1:])
				buf = buf[:n-1]
			}
			buf = append(buf, v)
			if len(buf) == n && !yield(slices.Clone(buf)) {
				return
			}
		}
	}}
}

// Scan returns a Stream of the running results of combining the elements of s from
// left to right, starting from initial. It is like Fold, but yields every
// intermediate result instead of only the last one, and works on infinite streams.
// The output has one element per input element and does not include initial.
//
//	stream.Scan(stream.Of(1, 2, 3), 0, func(sum, v int) int { return sum + v }) // 1 3 6
func Scan[T, A any](s Stream[T], initial A, accumulator func(acc A, v T) A) Stream[A] {
	checkFunc(accumulator == nil, "Scan")
	src := s.All()
	return Stream[A]{size: s.size, seq: func(yield func(A) bool) {
		acc := initial
		for v := range src {
			acc = accumulator(acc, v)
			if !yield(acc) {
				return
			}
		}
	}}
}

// capFor returns the capacity to allocate for a buffer of up to n elements drawn
// from a stream of the given size, which may be -1 if unknown.
func capFor(n, size int) int {
	if size >= 0 {
		return min(n, size)
	}
	return n
}

// Join concatenates the elements of s, placing sep between them.
func Join[T ~string](s Stream[T], sep string) string {
	var b strings.Builder
	first := true
	for v := range s.All() {
		if !first {
			b.WriteString(sep)
		}
		first = false
		b.WriteString(string(v))
	}
	return b.String()
}
