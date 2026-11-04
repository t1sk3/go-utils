package stream

import "cmp"

// This file holds helpers that build comparators for Sorted, Min and Max:
//
//	byAge := stream.Comparing(func(u User) int { return u.Age })
//	users.Sorted(stream.Reversed(byAge)) // oldest first
//
// They produce ordinary comparison functions, so they also work with the standard
// library, such as slices.SortFunc.

// Comparing returns a comparator that orders elements by the natural order of the
// value that key extracts from them. key runs for both elements on every comparison,
// so keep it cheap.
func Comparing[T any, K cmp.Ordered](key func(T) K) func(a, b T) int {
	checkFunc(key == nil, "Comparing")
	return func(a, b T) int { return cmp.Compare(key(a), key(b)) }
}

// Reversed returns a comparator with the opposite order to comparator. Sorted is
// stable, and that still holds for a reversed comparator: elements that compare equal
// keep their original relative order.
func Reversed[T any](comparator func(a, b T) int) func(a, b T) int {
	checkFunc(comparator == nil, "Reversed")
	return func(a, b T) int { return comparator(b, a) }
}

// ThenComparing returns a comparator that orders elements by first and breaks ties
// with second, which is only called when first reports them equal.
func ThenComparing[T any](first, second func(a, b T) int) func(a, b T) int {
	checkFunc(first == nil || second == nil, "ThenComparing")
	return func(a, b T) int {
		if c := first(a, b); c != 0 {
			return c
		}
		return second(a, b)
	}
}
