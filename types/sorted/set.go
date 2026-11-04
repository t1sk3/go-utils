package sorted

import (
	"cmp"
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Set holds distinct values ordered by a comparator. See the package documentation.
type Set[T any] struct {
	m *Map[T, struct{}]
}

// NewSet returns a Set ordered by compare and holding the given values. It panics if
// compare is nil.
func NewSet[T any](compare func(a, b T) int, values ...T) *Set[T] {
	if compare == nil {
		panic("sorted: NewSet requires a non-nil comparator")
	}
	s := &Set[T]{m: NewMap[T, struct{}](compare)}
	s.Add(values...)
	return s
}

// NewOrderedSet returns a Set ordered by the natural order of its values and holding
// the given values.
func NewOrderedSet[T cmp.Ordered](values ...T) *Set[T] {
	return NewSet(cmp.Compare[T], values...)
}

// Add adds the given values. A value that compares equal to one already in s is
// ignored, so the existing value stays.
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

// Clone returns a copy of s with the same comparator that does not share storage with
// it. Values are copied by assignment.
func (s *Set[T]) Clone() *Set[T] {
	return &Set[T]{m: s.m.Clone()}
}

// First returns the smallest value, or an empty Optional if s is empty.
func (s *Set[T]) First() optional.Optional[T] {
	return key(s.m.First())
}

// Last returns the largest value, or an empty Optional if s is empty.
func (s *Set[T]) Last() optional.Optional[T] {
	return key(s.m.Last())
}

// PopFirst removes and returns the smallest value, or returns an empty Optional if s
// is empty.
func (s *Set[T]) PopFirst() optional.Optional[T] {
	return key(s.m.PopFirst())
}

// PopLast removes and returns the largest value, or returns an empty Optional if s is
// empty.
func (s *Set[T]) PopLast() optional.Optional[T] {
	return key(s.m.PopLast())
}

// Floor returns the largest value less than or equal to v, or an empty Optional if
// there is none.
func (s *Set[T]) Floor(v T) optional.Optional[T] {
	return key(s.m.Floor(v))
}

// Ceiling returns the smallest value greater than or equal to v, or an empty Optional
// if there is none.
func (s *Set[T]) Ceiling(v T) optional.Optional[T] {
	return key(s.m.Ceiling(v))
}

// Lower returns the largest value strictly less than v, or an empty Optional if there
// is none.
func (s *Set[T]) Lower(v T) optional.Optional[T] {
	return key(s.m.Lower(v))
}

// Higher returns the smallest value strictly greater than v, or an empty Optional if
// there is none.
func (s *Set[T]) Higher(v T) optional.Optional[T] {
	return key(s.m.Higher(v))
}

// All returns an iterator over the values in ascending order. s must not be modified
// during iteration.
func (s *Set[T]) All() iter.Seq[T] {
	return s.m.Keys()
}

// Backward returns an iterator over the values in descending order. s must not be
// modified during iteration.
func (s *Set[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.m.root.descend(func(v T, _ struct{}) bool { return yield(v) })
	}
}

// Range returns an iterator over the values from from (inclusive) to to (exclusive),
// in ascending order. It is empty if from is not less than to. s must not be modified
// during iteration.
func (s *Set[T]) Range(from, to T) iter.Seq[T] {
	return func(yield func(T) bool) {
		s.m.ascendRange(s.m.root, from, to, func(v T, _ struct{}) bool { return yield(v) })
	}
}

// String formats the values in order, for example "Set[1 2 3]".
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
