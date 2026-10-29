package stream

import (
	"cmp"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/optional"
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
func Sum[T Number](s Stream[T]) T {
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
