package stream

import (
	"iter"
	"slices"

	"github.com/t1sk3/go-utils/types/pair"
)

// This file holds the Stream methods that introduce a type parameter of their own,
// which Go allows since Go 1.27. Map is in stream.go with the other intermediate
// operations.
//
// Operations that constrain T (Distinct, Sort, Sum, Average, Summarize, Min, Max, ToSet,
// ToSeq2 and Join) are package-level functions in functions.go, because a method cannot
// add a constraint to its receiver's type parameter.
//
// Chunk, Window, Enumerate and Zip return a Stream of a type built from T, such as
// []T. Written directly, as Chunk(n int) Stream[[]T], the compiler rejects that as an
// instantiation cycle: Stream[T] would need Stream[[]T], whose Chunk would need
// Stream[[][]T], and so on. They therefore name the element type with a type parameter
// of their own whose constraint is exactly that type, such as [S []T]. The constraint
// has a single type, so Go infers S and callers never write it. For the same reason
// their bodies must not call each other or a function that returns Stream[[]T].

// FlatMap returns a Stream of the elements of each stream produced by applying mapper
// to the elements of s.
func (s Stream[T]) FlatMap[R any](mapper func(T) Stream[R]) Stream[R] {
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

// DistinctBy returns a Stream of the elements whose key is not shared by an earlier
// element, keeping the first occurrence of each key. Use it for element types that
// are not comparable; for comparable ones, the Distinct function is shorter.
func (s Stream[T]) DistinctBy[K comparable](key func(T) K) Stream[T] {
	checkFunc(key == nil, "DistinctBy")
	src := s.All()
	hint := sizeHint(s.size)
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

// Fold combines the elements from left to right, starting from initial. Unlike
// Reduce, the result may have a different type than the elements:
//
//	totalLen := words.Fold(0, func(n int, w string) int { return n + len(w) })
func (s Stream[T]) Fold[A any](initial A, accumulator func(acc A, v T) A) A {
	checkFunc(accumulator == nil, "Fold")
	acc := initial
	for v := range s.All() {
		acc = accumulator(acc, v)
	}
	return acc
}

// Scan returns a Stream of the running results of combining the elements from left to
// right, starting from initial. It is like Fold, but yields every intermediate result
// instead of only the last one, and works on infinite streams. The output has one
// element per input element and does not include initial.
//
//	stream.Of(1, 2, 3).Scan(0, func(sum, v int) int { return sum + v }) // 1 3 6
func (s Stream[T]) Scan[A any](initial A, accumulator func(acc A, v T) A) Stream[A] {
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

// GroupBy collects the elements into a map from key to the elements with that key,
// preserving their order.
func (s Stream[T]) GroupBy[K comparable](key func(T) K) map[K][]T {
	checkFunc(key == nil, "GroupBy")
	groups := make(map[K][]T)
	for v := range s.All() {
		k := key(v)
		groups[k] = append(groups[k], v)
	}
	return groups
}

// CountBy returns the number of elements for each key.
func (s Stream[T]) CountBy[K comparable](key func(T) K) map[K]int {
	checkFunc(key == nil, "CountBy")
	counts := make(map[K]int)
	for v := range s.All() {
		counts[key(v)]++
	}
	return counts
}

// ToMap collects the elements into a map. If several elements produce the same key,
// the last one wins. Use ToMapMerge to combine them instead:
//
//	lengths := words.ToMap(stream.Identity, func(w string) int { return len(w) })
func (s Stream[T]) ToMap[K comparable, V any](key func(T) K, value func(T) V) map[K]V {
	checkFunc(key == nil || value == nil, "ToMap")
	m := make(map[K]V, sizeHint(s.size))
	for v := range s.All() {
		m[key(v)] = value(v)
	}
	return m
}

// ToMapMerge collects the elements into a map. If several elements produce the same
// key, their values are combined from left to right with merge:
//
//	totals := orders.ToMapMerge(Order.Customer, Order.Amount,
//		func(a, b int) int { return a + b })
func (s Stream[T]) ToMapMerge[K comparable, V any](key func(T) K, value func(T) V, merge func(existing, next V) V) map[K]V {
	checkFunc(key == nil || value == nil || merge == nil, "ToMapMerge")
	m := make(map[K]V, sizeHint(s.size))
	for v := range s.All() {
		k, val := key(v), value(v)
		if existing, ok := m[k]; ok {
			val = merge(existing, val)
		}
		m[k] = val
	}
	return m
}

// Collect reduces the elements with c.
func (s Stream[T]) Collect[R any](c Collector[T, R]) R {
	checkFunc(c.start == nil, "Collect")
	add, finish := c.start()
	for v := range s.All() {
		add(v)
	}
	return finish()
}

// MapConcurrent applies mapper to the elements on at most n goroutines at once and
// returns the results in order. It is shorthand for one parallel stage, the
// counterpart of Java's Gatherers.mapConcurrent:
//
//	pages := urls.MapConcurrent(8, fetch)
//
// See ParallelStream for what this guarantees. MapConcurrent panics if n is not
// positive.
func (s Stream[T]) MapConcurrent[R any](n int, mapper func(T) R) Stream[R] {
	checkFunc(mapper == nil, "MapConcurrent")
	return s.Parallel(n).Map(mapper).Sequential()
}

// Enumerate returns a Stream that pairs each element with its index, starting at 0.
// P is always pair.Pair[int, T] and is inferred.
func (s Stream[T]) Enumerate[P pair.Pair[int, T]]() Stream[P] {
	src := s.All()
	return Stream[P]{size: s.size, seq: func(yield func(P) bool) {
		i := 0
		for v := range src {
			if !yield(P(pair.Of(i, v))) {
				return
			}
			i++
		}
	}}
}

// Zip returns a Stream that pairs the elements of s and b by position. It ends when
// either stream ends, so zipping with an infinite stream is fine. P is always
// pair.Pair[T, B] and is inferred.
//
// When b comes straight from FromSlice or Of, Zip indexes it directly; otherwise it
// has to step b with iter.Pull, which is roughly ten times slower per element. If
// only one side is a plain slice, pass it as b.
//
// Zip reads an element of s before the matching element of b. When b is not a plain
// slice and ends first, one element of s has therefore been read and is dropped, which
// matters only if reading s has side effects. A pair needs both elements, so one side
// always has to be read without knowing whether the other has one left.
func (s Stream[T]) Zip[B any, P pair.Pair[T, B]](b Stream[B]) Stream[P] {
	size := -1
	if s.size >= 0 && b.size >= 0 {
		size = min(s.size, b.size)
	}
	srcA := s.All()
	if b.seq == nil {
		// b is a plain slice: index it instead of starting a coroutine with iter.Pull.
		bs := b.slice
		return Stream[P]{size: size, seq: func(yield func(P) bool) {
			if len(bs) == 0 {
				return
			}
			i := 0
			for va := range srcA {
				if !yield(P(pair.Of(va, bs[i]))) {
					return
				}
				if i++; i == len(bs) {
					return
				}
			}
		}}
	}
	srcB := b.seq
	return Stream[P]{size: size, seq: func(yield func(P) bool) {
		nextB, stop := iter.Pull(srcB)
		defer stop()
		for va := range srcA {
			vb, ok := nextB()
			if !ok || !yield(P(pair.Of(va, vb))) {
				return
			}
		}
	}}
}

// Chunk returns a Stream of consecutive, non-overlapping slices of n elements. The
// last chunk is shorter if the elements do not divide evenly. Each chunk is a new
// slice that the caller may keep or modify. S is always []T and is inferred. Chunk
// panics if n is not positive.
//
//	stream.Range(0, 5).Chunk(2) // [0 1] [2 3] [4]
func (s Stream[T]) Chunk[S []T](n int) Stream[S] {
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
	return Stream[S]{size: size, seq: func(yield func(S) bool) {
		var buf []T
		for v := range src {
			if buf == nil {
				buf = make([]T, 0, capacity)
			}
			buf = append(buf, v)
			if len(buf) == n {
				if !yield(S(buf)) {
					return
				}
				buf = nil
			}
		}
		if len(buf) > 0 {
			yield(S(buf))
		}
	}}
}

// Window returns a Stream of every run of n consecutive elements, advancing one
// element at a time. A stream with fewer than n elements produces no windows. Each
// window is a new slice that the caller may keep or modify. S is always []T and is
// inferred. Window panics if n is not positive.
//
//	stream.Range(0, 4).Window(2) // [0 1] [1 2] [2 3]
func (s Stream[T]) Window[S []T](n int) Stream[S] {
	if n <= 0 {
		panic("stream: Window requires a positive size")
	}
	size := -1
	if s.size >= 0 {
		size = max(s.size-n+1, 0)
	}
	src := s.All()
	capacity := capFor(n, s.size)
	return Stream[S]{size: size, seq: func(yield func(S) bool) {
		var buf []T
		for v := range src {
			if buf == nil {
				buf = make([]T, 0, capacity)
			}
			if len(buf) == n {
				copy(buf, buf[1:])
				buf = buf[:n-1]
			}
			buf = append(buf, v)
			if len(buf) == n && !yield(S(slices.Clone(buf))) {
				return
			}
		}
	}}
}

// maxPrealloc is the most elements a buffer is sized for before any have arrived. The
// stream's size and the caller's n say how many elements may come, not how many will:
// a later Limit can stop after three, distinct keys can be few, and n can be far larger
// than the stream. Buffers start no bigger than this and grow as elements arrive.
const maxPrealloc = 1024

// sizeHint returns the capacity to allocate up front for a collection that receives at
// most one entry per element of a stream of the given size, which may be -1 if unknown.
func sizeHint(size int) int {
	return min(max(size, 0), maxPrealloc)
}

// capFor returns the capacity to allocate up front for a buffer of up to n elements
// drawn from a stream of the given size, which may be -1 if unknown.
func capFor(n, size int) int {
	if size >= 0 {
		n = min(n, size)
	}
	return min(n, maxPrealloc)
}
