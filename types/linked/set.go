package linked

import (
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Set is a hash set that iterates in insertion order. See the package documentation.
type Set[T comparable] struct {
	m Map[T, struct{}]
}

// NewSet returns a Set holding the given values, in the order of their first occurrence.
func NewSet[T comparable](values ...T) *Set[T] {
	s := &Set[T]{}
	s.Add(values...)
	return s
}

// CollectSet returns a Set holding the values of seq, in the order of their first
// occurrence.
func CollectSet[T comparable](seq iter.Seq[T]) *Set[T] {
	s := &Set[T]{}
	for v := range seq {
		s.Add(v)
	}
	return s
}

// Add adds the given values to the back. A value already in s keeps its position.
func (s *Set[T]) Add(values ...T) {
	for _, v := range values {
		s.m.Put(v, struct{}{})
	}
}

// Remove removes the given values. Values that are not in s are ignored.
func (s *Set[T]) Remove(values ...T) {
	for _, v := range values {
		s.m.Remove(v)
	}
}

// Has reports whether v is in s.
func (s *Set[T]) Has(v T) bool {
	return s.m.Has(v)
}

// MoveToBack makes v the last value and reports whether it was present.
func (s *Set[T]) MoveToBack(v T) bool {
	return s.m.MoveToBack(v)
}

// MoveToFront makes v the first value and reports whether it was present.
func (s *Set[T]) MoveToFront(v T) bool {
	return s.m.MoveToFront(v)
}

// Len returns the number of values in s.
func (s *Set[T]) Len() int {
	return s.m.Len()
}

// IsEmpty reports whether s has no values.
func (s *Set[T]) IsEmpty() bool {
	return s.m.IsEmpty()
}

// Clear removes every value from s.
func (s *Set[T]) Clear() {
	s.m.Clear()
}

// Clone returns a copy of s in the same order that does not share storage with it.
// Values are copied by assignment.
func (s *Set[T]) Clone() *Set[T] {
	c := &Set[T]{}
	for v := range s.All() {
		c.Add(v)
	}
	return c
}

// First returns the oldest value, or an empty Optional if s is empty.
func (s *Set[T]) First() optional.Optional[T] {
	return key(s.m.First())
}

// Last returns the newest value, or an empty Optional if s is empty.
func (s *Set[T]) Last() optional.Optional[T] {
	return key(s.m.Last())
}

// PopFirst removes and returns the oldest value, or returns an empty Optional if s is
// empty.
func (s *Set[T]) PopFirst() optional.Optional[T] {
	return key(s.m.PopFirst())
}

// PopLast removes and returns the newest value, or returns an empty Optional if s is
// empty.
func (s *Set[T]) PopLast() optional.Optional[T] {
	return key(s.m.PopLast())
}

// All returns an iterator over the values from oldest to newest. The current value may
// be removed during iteration; any other change makes the iteration unspecified.
func (s *Set[T]) All() iter.Seq[T] {
	return s.m.Keys()
}

// Backward returns an iterator over the values from newest to oldest. The current
// value may be removed during iteration; any other change makes the iteration
// unspecified.
func (s *Set[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s.m.Backward() {
			if !yield(v) {
				return
			}
		}
	}
}

// String formats the values from oldest to newest, for example "Set[b a]".
func (s *Set[T]) String() string {
	var b strings.Builder
	b.WriteString("Set[")
	first := true
	for v := range s.All() {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprint(&b, v)
	}
	b.WriteByte(']')
	return b.String()
}

func key[T any](e optional.Optional[pair.Pair[T, struct{}]]) optional.Optional[T] {
	if p, ok := e.Value(); ok {
		return optional.Of(p.First)
	}
	return optional.Empty[T]()
}
