package stream

import (
	"strings"

	"github.com/t1sk3/go-utils/types/constraints"
	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/set"
)

// This file holds collectors, the counterparts of Java's Collectors. Their strength is
// that they nest, so a grouping can reduce each group instead of keeping its elements:
//
//	averageAge := stream.Collect(users, stream.GroupingBy(User.Team, stream.Averaging(User.Age)))
//
// For a single reduction, functions such as GroupBy, CountBy, Sum and Join are shorter
// and faster. Go cannot infer a collector's element type from where it is used, so the
// collectors without a function argument need it spelled out: Counting[User]().

// Collector describes how to reduce elements of type T to a result of type R. Build one
// with the functions in this package or NewCollector, and run it with Collect. A
// Collector holds no state of its own, so it can be used any number of times.
type Collector[T, R any] struct {
	start func() (add func(T), finish func() R)
}

// NewCollector returns a Collector for a reduction that the built-in collectors do not
// cover. Each run calls supplier for a new accumulation, passes it through accumulator
// once per element, and converts the last one to the result with finisher.
func NewCollector[T, A, R any](supplier func() A, accumulator func(acc A, v T) A, finisher func(A) R) Collector[T, R] {
	checkFunc(supplier == nil || accumulator == nil || finisher == nil, "NewCollector")
	return Collector[T, R]{start: func() (func(T), func() R) {
		acc := supplier()
		return func(v T) { acc = accumulator(acc, v) }, func() R { return finisher(acc) }
	}}
}

// Collect reduces the elements of s with c.
func Collect[T, R any](s Stream[T], c Collector[T, R]) R {
	checkFunc(c.start == nil, "Collect")
	add, finish := c.start()
	for v := range s.All() {
		add(v)
	}
	return finish()
}

// IntoSlice returns a Collector of the elements into a slice, in order. The result is
// never nil.
func IntoSlice[T any]() Collector[T, []T] {
	return Collector[T, []T]{start: func() (func(T), func() []T) {
		out := []T{}
		return func(v T) { out = append(out, v) }, func() []T { return out }
	}}
}

// IntoSet returns a Collector of the elements into a set.Set.
func IntoSet[T comparable]() Collector[T, set.Set[T]] {
	return Collector[T, set.Set[T]]{start: func() (func(T), func() set.Set[T]) {
		out := set.Of[T]()
		return func(v T) { out.Add(v) }, func() set.Set[T] { return out }
	}}
}

// Joining returns a Collector that concatenates the elements, placing sep between them.
func Joining[T ~string](sep string) Collector[T, string] {
	return Collector[T, string]{start: func() (func(T), func() string) {
		var b strings.Builder
		first := true
		return func(v T) {
			if !first {
				b.WriteString(sep)
			}
			first = false
			b.WriteString(string(v))
		}, b.String
	}}
}

// Counting returns a Collector of the number of elements.
func Counting[T any]() Collector[T, int] {
	return Collector[T, int]{start: func() (func(T), func() int) {
		n := 0
		return func(T) { n++ }, func() int { return n }
	}}
}

// Summing returns a Collector of the sum of value applied to each element.
func Summing[T any, N constraints.Number](value func(T) N) Collector[T, N] {
	checkFunc(value == nil, "Summing")
	return Collector[T, N]{start: func() (func(T), func() N) {
		var total N
		return func(v T) { total += value(v) }, func() N { return total }
	}}
}

// Averaging returns a Collector of the arithmetic mean of value applied to each
// element, or an empty Optional if there are no elements. Values are summed as float64,
// so very large integers may lose precision.
func Averaging[T any, N constraints.Number](value func(T) N) Collector[T, optional.Optional[float64]] {
	checkFunc(value == nil, "Averaging")
	return Collector[T, optional.Optional[float64]]{start: func() (func(T), func() optional.Optional[float64]) {
		var total float64
		n := 0
		return func(v T) {
				total += float64(value(v))
				n++
			}, func() optional.Optional[float64] {
				if n == 0 {
					return optional.Empty[float64]()
				}
				return optional.Of(total / float64(n))
			}
	}}
}

// MinBy returns a Collector of the smallest element according to comparator, or an
// empty Optional if there are no elements. If several elements are equally small, the
// first is kept.
func MinBy[T any](comparator func(a, b T) int) Collector[T, optional.Optional[T]] {
	checkFunc(comparator == nil, "MinBy")
	return bestBy(func(v, best T) bool { return comparator(v, best) < 0 })
}

// MaxBy returns a Collector of the largest element according to comparator, or an empty
// Optional if there are no elements. If several elements are equally large, the first
// is kept.
func MaxBy[T any](comparator func(a, b T) int) Collector[T, optional.Optional[T]] {
	checkFunc(comparator == nil, "MaxBy")
	return bestBy(func(v, best T) bool { return comparator(v, best) > 0 })
}

func bestBy[T any](better func(v, best T) bool) Collector[T, optional.Optional[T]] {
	return Collector[T, optional.Optional[T]]{start: func() (func(T), func() optional.Optional[T]) {
		var best optional.Optional[T]
		return func(v T) {
			if b, ok := best.Value(); !ok || better(v, b) {
				best = optional.Of(v)
			}
		}, func() optional.Optional[T] { return best }
	}}
}

// Folding returns a Collector that combines the elements from left to right, starting
// from initial. It is the Collector form of the Fold function.
func Folding[T, A any](initial A, accumulator func(acc A, v T) A) Collector[T, A] {
	checkFunc(accumulator == nil, "Folding")
	return Collector[T, A]{start: func() (func(T), func() A) {
		acc := initial
		return func(v T) { acc = accumulator(acc, v) }, func() A { return acc }
	}}
}

// GroupingBy returns a Collector that divides the elements by key and reduces each
// group with downstream. Only keys that occur appear in the result.
func GroupingBy[T any, K comparable, R any](key func(T) K, downstream Collector[T, R]) Collector[T, map[K]R] {
	checkFunc(key == nil, "GroupingBy")
	checkFunc(downstream.start == nil, "GroupingBy")
	return Collector[T, map[K]R]{start: func() (func(T), func() map[K]R) {
		groups := make(map[K]collectorRun[T, R])
		return func(v T) {
				k := key(v)
				g, ok := groups[k]
				if !ok {
					g.add, g.finish = downstream.start()
					groups[k] = g
				}
				g.add(v)
			}, func() map[K]R {
				out := make(map[K]R, len(groups))
				for k, g := range groups {
					out[k] = g.finish()
				}
				return out
			}
	}}
}

// PartitioningBy returns a Collector that divides the elements by predicate and reduces
// each side with downstream. The result always holds both true and false, even when a
// side has no elements.
func PartitioningBy[T, R any](predicate func(T) bool, downstream Collector[T, R]) Collector[T, map[bool]R] {
	checkFunc(predicate == nil, "PartitioningBy")
	checkFunc(downstream.start == nil, "PartitioningBy")
	return Collector[T, map[bool]R]{start: func() (func(T), func() map[bool]R) {
		addTrue, finishTrue := downstream.start()
		addFalse, finishFalse := downstream.start()
		return func(v T) {
				if predicate(v) {
					addTrue(v)
				} else {
					addFalse(v)
				}
			}, func() map[bool]R {
				return map[bool]R{true: finishTrue(), false: finishFalse()}
			}
	}}
}

// Mapping returns a Collector that applies mapper to each element before passing it to
// downstream.
func Mapping[T, U, R any](mapper func(T) U, downstream Collector[U, R]) Collector[T, R] {
	checkFunc(mapper == nil, "Mapping")
	checkFunc(downstream.start == nil, "Mapping")
	return Collector[T, R]{start: func() (func(T), func() R) {
		add, finish := downstream.start()
		return func(v T) { add(mapper(v)) }, finish
	}}
}

// FlatMapping returns a Collector that passes every element of the stream that mapper
// produces for each element to downstream.
func FlatMapping[T, U, R any](mapper func(T) Stream[U], downstream Collector[U, R]) Collector[T, R] {
	checkFunc(mapper == nil, "FlatMapping")
	checkFunc(downstream.start == nil, "FlatMapping")
	return Collector[T, R]{start: func() (func(T), func() R) {
		add, finish := downstream.start()
		return func(v T) {
			for u := range mapper(v).All() {
				add(u)
			}
		}, finish
	}}
}

// Filtering returns a Collector that passes only the elements that satisfy predicate to
// downstream. Unlike Stream.Filter before GroupingBy, it keeps groups that end up empty.
func Filtering[T, R any](predicate func(T) bool, downstream Collector[T, R]) Collector[T, R] {
	checkFunc(predicate == nil, "Filtering")
	checkFunc(downstream.start == nil, "Filtering")
	return Collector[T, R]{start: func() (func(T), func() R) {
		add, finish := downstream.start()
		return func(v T) {
			if predicate(v) {
				add(v)
			}
		}, finish
	}}
}

// CollectingAndThen returns a Collector that converts the result of downstream with
// finisher.
func CollectingAndThen[T, R, S any](downstream Collector[T, R], finisher func(R) S) Collector[T, S] {
	checkFunc(finisher == nil, "CollectingAndThen")
	checkFunc(downstream.start == nil, "CollectingAndThen")
	return Collector[T, S]{start: func() (func(T), func() S) {
		add, finish := downstream.start()
		return add, func() S { return finisher(finish()) }
	}}
}

// Teeing returns a Collector that passes every element to both first and second, then
// combines their results with merge. It computes two results in one pass:
//
//	spread := stream.Collect(prices, stream.Teeing(
//		stream.MinBy(cmp.Compare[int]), stream.MaxBy(cmp.Compare[int]), newRange))
func Teeing[T, R1, R2, R any](first Collector[T, R1], second Collector[T, R2], merge func(R1, R2) R) Collector[T, R] {
	checkFunc(merge == nil, "Teeing")
	checkFunc(first.start == nil || second.start == nil, "Teeing")
	return Collector[T, R]{start: func() (func(T), func() R) {
		add1, finish1 := first.start()
		add2, finish2 := second.start()
		return func(v T) {
				add1(v)
				add2(v)
			}, func() R {
				return merge(finish1(), finish2())
			}
	}}
}

// collectorRun is one run of a Collector.
type collectorRun[T, R any] struct {
	add    func(T)
	finish func() R
}
