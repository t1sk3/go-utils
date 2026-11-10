package stream

// This file holds the Stream methods that introduce a type parameter of their own,
// which Go allows since Go 1.27. Map is in stream.go with the other intermediate
// operations.
//
// Operations that constrain T (Distinct, Sort, Sum, Average, Summarize, Min, Max, ToSet,
// ToSeq2 and Join) are package-level functions in functions.go, because a method cannot
// add a constraint to its receiver's type parameter. So are Enumerate, Chunk, Window
// and Zip: a method of Stream[T] that returns a Stream of a type built from T, such as
// Stream[[]T], is rejected by the compiler as an instantiation cycle.

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
	m := make(map[K]V, max(s.size, 0))
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
	m := make(map[K]V, max(s.size, 0))
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
