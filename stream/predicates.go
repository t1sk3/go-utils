package stream

import (
	"cmp"
	"slices"

	"github.com/t1sk3/go-utils/types/constraints"
	"github.com/t1sk3/go-utils/types/set"
)

// This file holds ready-made predicates for Filter, TakeWhile, DropWhile,
// AnyMatch, AllMatch, NoneMatch and Partition, plus combinators to build new ones.
//
// The plain predicates can be passed directly and Go infers their type:
//
//	stream.Range(0, 10).Filter(stream.IsEven) // 0 2 4 6 8
//
// The constructors (GreaterThan, Between, In, ...) return a predicate:
//
//	stream.FromSlice(ages).Filter(stream.AtLeast(18))
//
// When a generic predicate is passed to a combinator such as Not, Go cannot infer
// its type, so instantiate it explicitly:
//
//	stream.Range(0, 10).Filter(stream.Not(stream.IsEven[int])) // 1 3 5 7 9

// IsEven reports whether v is divisible by 2.
func IsEven[T constraints.Integer](v T) bool {
	return v%2 == 0
}

// IsOdd reports whether v is not divisible by 2. It is equivalent to Not(IsEven).
func IsOdd[T constraints.Integer](v T) bool {
	return v%2 != 0
}

// IsPositive reports whether v is greater than zero.
func IsPositive[T constraints.Number](v T) bool {
	return v > 0
}

// IsNegative reports whether v is less than zero.
func IsNegative[T constraints.Number](v T) bool {
	return v < 0
}

// IsZero reports whether v is the zero value of its type, such as 0, "" or nil.
// Use Not(IsZero[T]) to keep only non-zero values.
func IsZero[T comparable](v T) bool {
	var zero T
	return v == zero
}

// Equal returns a predicate that reports whether its argument equals target.
func Equal[T comparable](target T) func(T) bool {
	return func(v T) bool { return v == target }
}

// GreaterThan returns a predicate that reports whether its argument is greater
// than bound.
func GreaterThan[T cmp.Ordered](bound T) func(T) bool {
	return func(v T) bool { return v > bound }
}

// LessThan returns a predicate that reports whether its argument is less than bound.
func LessThan[T cmp.Ordered](bound T) func(T) bool {
	return func(v T) bool { return v < bound }
}

// AtLeast returns a predicate that reports whether its argument is greater than or
// equal to bound.
func AtLeast[T cmp.Ordered](bound T) func(T) bool {
	return func(v T) bool { return v >= bound }
}

// AtMost returns a predicate that reports whether its argument is less than or
// equal to bound.
func AtMost[T cmp.Ordered](bound T) func(T) bool {
	return func(v T) bool { return v <= bound }
}

// Between returns a predicate that reports whether its argument lies in the closed
// interval [low, high].
func Between[T cmp.Ordered](low, high T) func(T) bool {
	return func(v T) bool { return v >= low && v <= high }
}

// In returns a predicate that reports whether its argument is one of values. As with
// a map key, NaN is never found, and the predicate panics for an interface value that
// holds something unhashable, such as a slice.
func In[T comparable](values ...T) func(T) bool {
	allowed := set.Of(values...)
	return allowed.Has
}

// Not returns a predicate that reports the opposite of predicate, so Not(IsEven[int])
// behaves like IsOdd.
func Not[T any](predicate func(T) bool) func(T) bool {
	checkFunc(predicate == nil, "Not")
	return func(v T) bool { return !predicate(v) }
}

// And returns a predicate that reports whether every one of predicates holds. It
// stops at the first that does not. With no predicates it always reports true.
func And[T any](predicates ...func(T) bool) func(T) bool {
	predicates = slices.Clone(predicates)
	for _, p := range predicates {
		checkFunc(p == nil, "And")
	}
	return func(v T) bool {
		for _, p := range predicates {
			if !p(v) {
				return false
			}
		}
		return true
	}
}

// Or returns a predicate that reports whether any one of predicates holds. It stops
// at the first that does. With no predicates it always reports false.
func Or[T any](predicates ...func(T) bool) func(T) bool {
	predicates = slices.Clone(predicates)
	for _, p := range predicates {
		checkFunc(p == nil, "Or")
	}
	return func(v T) bool {
		for _, p := range predicates {
			if p(v) {
				return true
			}
		}
		return false
	}
}
