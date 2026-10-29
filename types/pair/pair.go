// Package pair provides Pair, a value holding two values of possibly different types.
//
// Pair is what stream.Zip, stream.Enumerate and stream.FromMap produce, and it is the
// usual way to carry a key and a value, or two related results, through generic code
// that only has room for one type parameter.
package pair

import "fmt"

// Pair holds two values. It is a plain value type, so it is comparable whenever A and
// B are, and can be used as a map key.
type Pair[A, B any] struct {
	First  A
	Second B
}

// Of returns a Pair of a and b.
func Of[A, B any](a A, b B) Pair[A, B] {
	return Pair[A, B]{First: a, Second: b}
}

// Unpack returns both values, so a Pair can be destructured in one assignment:
//
//	name, age := p.Unpack()
func (p Pair[A, B]) Unpack() (A, B) {
	return p.First, p.Second
}

// Swap returns a Pair with the values in the opposite order.
func (p Pair[A, B]) Swap() Pair[B, A] {
	return Pair[B, A]{First: p.Second, Second: p.First}
}

func (p Pair[A, B]) String() string {
	return fmt.Sprintf("(%v, %v)", p.First, p.Second)
}
