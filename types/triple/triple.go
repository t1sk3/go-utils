// Package triple provides Triple, a value holding three values of possibly different
// types. It is the three-value counterpart of pair.Pair.
package triple

import "fmt"

// Triple holds three values. It is a plain value type, so it is comparable whenever A,
// B and C are, and can be used as a map key.
type Triple[A, B, C any] struct {
	First  A
	Second B
	Third  C
}

// Of returns a Triple of a, b and c.
func Of[A, B, C any](a A, b B, c C) Triple[A, B, C] {
	return Triple[A, B, C]{First: a, Second: b, Third: c}
}

// Unpack returns all three values, so a Triple can be destructured in one assignment:
//
//	x, y, z := t.Unpack()
func (t Triple[A, B, C]) Unpack() (A, B, C) {
	return t.First, t.Second, t.Third
}

func (t Triple[A, B, C]) String() string {
	return fmt.Sprintf("(%v, %v, %v)", t.First, t.Second, t.Third)
}
