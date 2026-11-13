// Package set provides Set, an unordered collection of distinct comparable values.
//
// The zero value is an empty Set ready to use:
//
//	var seen set.Set[string]
//	seen.Add("a", "b")
//
// Values are compared the way map keys are. A NaN is not equal to itself, so each one
// added is a separate value that Has does not find and Remove cannot remove; RemoveIf
// and Clear can. If T is an interface type, a value holding something unhashable, such
// as a slice, makes the method panic.
//
// Like a map, a Set refers to shared storage: copying a non-empty Set gives a second
// handle to the same elements. Use Clone for an independent copy. A Set is not safe
// for concurrent use without external locking.
package set

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

// Set is an unordered collection of distinct values. See the package documentation.
type Set[T comparable] struct {
	m map[T]struct{}
}

// Of returns a Set holding the given values.
func Of[T comparable](values ...T) Set[T] {
	s := Set[T]{m: make(map[T]struct{}, len(values))}
	for _, v := range values {
		s.m[v] = struct{}{}
	}
	return s
}

// Collect returns a Set holding the values of seq, such as maps.Keys(m) or
// slices.Values(s).
func Collect[T comparable](seq iter.Seq[T]) Set[T] {
	s := Set[T]{m: make(map[T]struct{})}
	for v := range seq {
		s.m[v] = struct{}{}
	}
	return s
}

// Add adds the given values to s and returns how many were not already present.
func (s *Set[T]) Add(values ...T) int {
	if s.m == nil {
		s.m = make(map[T]struct{}, len(values))
	}
	before := len(s.m)
	for _, v := range values {
		s.m[v] = struct{}{}
	}
	return len(s.m) - before
}

// AddSeq adds the values of seq to s and returns how many were not already present.
func (s *Set[T]) AddSeq(seq iter.Seq[T]) int {
	if s.m == nil {
		s.m = make(map[T]struct{})
	}
	before := len(s.m)
	for v := range seq {
		s.m[v] = struct{}{}
	}
	return len(s.m) - before
}

// Pop removes and returns some value of s, or returns an empty Optional if s is empty.
// Which value it picks is unspecified. It is the way to work through a Set that grows
// while it is being processed, such as a set of items still to visit.
func (s *Set[T]) Pop() optional.Optional[T] {
	for v := range s.m {
		if v != v {
			// delete cannot remove a value that is not equal to itself, such as NaN,
			// so rebuild the map without this one occurrence of it.
			rebuilt := make(map[T]struct{}, len(s.m))
			skipped := false
			for u := range s.m {
				if u != u && !skipped {
					skipped = true
					continue
				}
				rebuilt[u] = struct{}{}
			}
			s.m = rebuilt
			return optional.Of(v)
		}
		delete(s.m, v)
		return optional.Of(v)
	}
	return optional.Empty[T]()
}

// Remove removes the given values from s and returns how many were present. Values
// that are not in s are ignored.
func (s *Set[T]) Remove(values ...T) int {
	before := len(s.m)
	for _, v := range values {
		delete(s.m, v)
	}
	return before - len(s.m)
}

// RemoveIf removes the values that satisfy predicate and returns how many it removed.
func (s *Set[T]) RemoveIf(predicate func(T) bool) int {
	before := len(s.m)
	// delete cannot remove a value that is not equal to itself, such as NaN. Those are
	// counted, and if any should go, the map is rebuilt without them.
	var keptUnequal []T
	removeUnequal := false
	for v := range s.m {
		switch match := predicate(v); {
		case v == v:
			if match {
				delete(s.m, v)
			}
		case match:
			removeUnequal = true
		default:
			keptUnequal = append(keptUnequal, v)
		}
	}
	if removeUnequal {
		rebuilt := make(map[T]struct{}, len(s.m))
		for v := range s.m {
			if v == v {
				rebuilt[v] = struct{}{}
			}
		}
		for _, v := range keptUnequal {
			rebuilt[v] = struct{}{}
		}
		s.m = rebuilt
	}
	return before - len(s.m)
}

// RetainAll removes the values that are not in other and returns how many it removed.
func (s *Set[T]) RetainAll(other Set[T]) int {
	return s.RemoveIf(func(v T) bool {
		_, ok := other.m[v]
		return !ok
	})
}

// Clear removes every value from s.
func (s *Set[T]) Clear() {
	clear(s.m)
}

// Has reports whether v is in s.
func (s Set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

// HasAll reports whether every one of values is in s. It is true for no values.
func (s Set[T]) HasAll(values ...T) bool {
	for _, v := range values {
		if _, ok := s.m[v]; !ok {
			return false
		}
	}
	return true
}

// HasAny reports whether at least one of values is in s. It is false for no values.
func (s Set[T]) HasAny(values ...T) bool {
	for _, v := range values {
		if _, ok := s.m[v]; ok {
			return true
		}
	}
	return false
}

// ToSlice returns the values in a new slice, in no particular order. The result is
// never nil. For a sorted slice, use slices.Sorted(s.All()).
func (s Set[T]) ToSlice() []T {
	out := make([]T, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}
	return out
}

// Len returns the number of values in s.
func (s Set[T]) Len() int {
	return len(s.m)
}

// IsEmpty reports whether s has no values.
func (s Set[T]) IsEmpty() bool {
	return len(s.m) == 0
}

// All returns an iterator over the values of s, in no particular order.
func (s Set[T]) All() iter.Seq[T] {
	return maps.Keys(s.m)
}

// Clone returns a copy of s that does not share storage with it.
func (s Set[T]) Clone() Set[T] {
	return Set[T]{m: maps.Clone(s.m)}
}

// Union returns a new Set of the values that are in s, other, or both.
func (s Set[T]) Union(other Set[T]) Set[T] {
	out := Set[T]{m: make(map[T]struct{}, max(len(s.m), len(other.m)))}
	maps.Copy(out.m, s.m)
	maps.Copy(out.m, other.m)
	return out
}

// Intersection returns a new Set of the values that are in both s and other.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	small, large := s.m, other.m
	if len(small) > len(large) {
		small, large = large, small
	}
	out := Set[T]{m: make(map[T]struct{})}
	for v := range small {
		if _, ok := large[v]; ok {
			out.m[v] = struct{}{}
		}
	}
	return out
}

// Difference returns a new Set of the values that are in s but not in other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	out := Set[T]{m: make(map[T]struct{})}
	for v := range s.m {
		if _, ok := other.m[v]; !ok {
			out.m[v] = struct{}{}
		}
	}
	return out
}

// SymmetricDifference returns a new Set of the values that are in exactly one of s
// and other.
func (s Set[T]) SymmetricDifference(other Set[T]) Set[T] {
	out := s.Difference(other)
	for v := range other.m {
		if _, ok := s.m[v]; !ok {
			out.m[v] = struct{}{}
		}
	}
	return out
}

// IsSubsetOf reports whether every value of s is also in other.
func (s Set[T]) IsSubsetOf(other Set[T]) bool {
	if len(s.m) > len(other.m) {
		return false
	}
	for v := range s.m {
		if _, ok := other.m[v]; !ok {
			return false
		}
	}
	return true
}

// IsSupersetOf reports whether every value of other is also in s.
func (s Set[T]) IsSupersetOf(other Set[T]) bool {
	return other.IsSubsetOf(s)
}

// IsDisjoint reports whether s and other have no values in common.
func (s Set[T]) IsDisjoint(other Set[T]) bool {
	small, large := s.m, other.m
	if len(small) > len(large) {
		small, large = large, small
	}
	for v := range small {
		if _, ok := large[v]; ok {
			return false
		}
	}
	return true
}

// Equal reports whether s and other hold exactly the same values.
func (s Set[T]) Equal(other Set[T]) bool {
	return len(s.m) == len(other.m) && s.IsSubsetOf(other)
}

// String formats the values in sorted order of their %v representation, so output
// is stable across runs, for example "Set[a b c]".
func (s Set[T]) String() string {
	parts := make([]string, 0, len(s.m))
	for v := range s.m {
		parts = append(parts, fmt.Sprint(v))
	}
	slices.Sort(parts)
	return "Set[" + strings.Join(parts, " ") + "]"
}

// MarshalJSON encodes s as a JSON array. The elements are sorted by their encoded
// form, so the same Set always produces the same bytes, which keeps golden files,
// caches and diffs stable. That order is not numeric: 10 sorts before 9.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	encoded := make([][]byte, 0, len(s.m))
	size := 2
	for v := range s.m {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, b)
		size += len(b) + 1
	}
	slices.SortFunc(encoded, bytes.Compare)
	out := make([]byte, 0, size)
	out = append(out, '[')
	for i, b := range encoded {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, b...)
	}
	return append(out, ']'), nil
}

// UnmarshalJSON decodes a JSON array into s, replacing its contents. Duplicate
// values in the array are collapsed.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	var values []T
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*s = Of(values...)
	return nil
}
