package stream

import "github.com/t1sk3/go-utils/types/constraints"

// This file holds ready-made binary operators for Fold, Reduce, Scan and Folding.
// They can be passed directly and Go infers their type:
//
//	stream.Of(1, 2, 3, 4).Fold(0, stream.Add)      // 10
//	stream.Of(1, 2, 3, 4).Fold(1, stream.Multiply) // 24
//	stream.Of(1, 2, 3, 4).Reduce(stream.Multiply)  // Optional(24)
//
// They work for every numeric type, including floats and complex numbers. Integer
// results wrap around on overflow, as Go's + and * do.

// Add returns a + b. Folding with Add from 0 gives the same result as Sum.
func Add[T constraints.Numeric](a, b T) T {
	return a + b
}

// Multiply returns a * b. Fold with it from 1, the identity for multiplication;
// folding from 0 always gives 0.
func Multiply[T constraints.Numeric](a, b T) T {
	return a * b
}
