package stream

import "utils/optional"

// Stream is a lazy, chainable abstraction over a finite or potentially unbounded
// sequence of elements of type T. It supports intermediate operations that
// transform or filter the sequence and terminal operations that produce a result
// or side effects. Unless noted, intermediate operations are non-mutating and
// return a new Stream that preserves the encounter order of the source.

type Stream[T any] interface {
	Filter(predicate func(T) bool) Stream[T]

	Map(mapper func(T) any) Stream[any]

	ForEach(consumer func(T))

	FindFirst() optional.Optional[T]
	FindAny() optional.Optional[T]

	AllMatch(predicate func(T) bool) bool
	AnyMatch(predicate func(T) bool) bool
	NoneMatch(predicate func(T) bool) bool

	Distinct() Stream[T]

	Sorted(comparator func(T, T) int) Stream[T]

	Count() int64

	Min(comparator func(T, T) int) optional.Optional[T]
	Max(comparator func(T, T) int) optional.Optional[T]

	Limit(a int64) Stream[T]

	Skip(a int64) Stream[T]

	ToSlice() []T

	Reduce(f func(T, T) T) optional.Optional[T]
	ToStream() Stream[T]
}
