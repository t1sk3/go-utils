package stream

// This file holds the method forms of package functions that introduce a new type
// parameter, which Go allows on methods since Go 1.27. Each method forwards to the
// function of the same name, so the two forms behave identically.
//
// The functions that constrain T itself (Distinct, Sort, Sum, Average, Summarize, Min,
// Max, ToSet, ToSeq2 and Join) stay functions only, because a method cannot add a
// constraint to its receiver's type parameter. Enumerate, Chunk, Window and Zip stay
// functions only as well: a method of Stream[T] that returns a Stream of a type built
// from T, such as Stream[[]T], is rejected by the compiler as an instantiation cycle.

// FlatMap returns a Stream of the elements of each stream produced by applying mapper
// to the elements of s. It is the method form of the FlatMap function.
func (s Stream[T]) FlatMap[R any](mapper func(T) Stream[R]) Stream[R] {
	return FlatMap(s, mapper)
}

// DistinctBy returns a Stream of the elements whose key is not shared by an earlier
// element. It is the method form of the DistinctBy function.
func (s Stream[T]) DistinctBy[K comparable](key func(T) K) Stream[T] {
	return DistinctBy(s, key)
}

// Fold combines the elements from left to right, starting from initial. It is the
// method form of the Fold function.
func (s Stream[T]) Fold[A any](initial A, accumulator func(acc A, v T) A) A {
	return Fold(s, initial, accumulator)
}

// Scan returns a Stream of the running results of combining the elements from left to
// right, starting from initial. It is the method form of the Scan function.
func (s Stream[T]) Scan[A any](initial A, accumulator func(acc A, v T) A) Stream[A] {
	return Scan(s, initial, accumulator)
}

// GroupBy collects the elements into a map from key to the elements with that key,
// preserving their order. It is the method form of the GroupBy function.
func (s Stream[T]) GroupBy[K comparable](key func(T) K) map[K][]T {
	return GroupBy(s, key)
}

// CountBy returns the number of elements for each key. It is the method form of the
// CountBy function.
func (s Stream[T]) CountBy[K comparable](key func(T) K) map[K]int {
	return CountBy(s, key)
}

// ToMap collects the elements into a map, keeping the last value for a repeated key.
// It is the method form of the ToMap function.
func (s Stream[T]) ToMap[K comparable, V any](key func(T) K, value func(T) V) map[K]V {
	return ToMap(s, key, value)
}

// ToMapMerge collects the elements into a map, combining the values of a repeated key
// with merge. It is the method form of the ToMapMerge function.
func (s Stream[T]) ToMapMerge[K comparable, V any](key func(T) K, value func(T) V, merge func(existing, next V) V) map[K]V {
	return ToMapMerge(s, key, value, merge)
}

// Collect reduces the elements with c. It is the method form of the Collect function.
func (s Stream[T]) Collect[R any](c Collector[T, R]) R {
	return Collect(s, c)
}

// MapConcurrent applies mapper to the elements on at most n goroutines at once and
// returns the results in order. It is the method form of the MapConcurrent function.
func (s Stream[T]) MapConcurrent[R any](n int, mapper func(T) R) Stream[R] {
	return MapConcurrent(s, n, mapper)
}
