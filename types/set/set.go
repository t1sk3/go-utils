// Package set provides Set, an unordered collection of distinct comparable values.
//
// The zero value is an empty Set ready to use:
//
//	var seen set.Set[string]
//	seen.Add("a", "b")
//
// Like a map, a Set refers to shared storage: copying a non-empty Set gives a second
// handle to the same elements. Use Clone for an independent copy. A Set is not safe
// for concurrent use without external locking.
package set

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
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

// Add adds the given values to s.
func (s *Set[T]) Add(values ...T) {
	if s.m == nil {
		s.m = make(map[T]struct{}, len(values))
	}
	for _, v := range values {
		s.m[v] = struct{}{}
	}
}

// Remove removes the given values from s. Values that are not in s are ignored.
func (s *Set[T]) Remove(values ...T) {
	for _, v := range values {
		delete(s.m, v)
	}
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

// MarshalJSON encodes s as a JSON array, in no particular order.
func (s Set[T]) MarshalJSON() ([]byte, error) {
	values := make([]T, 0, len(s.m))
	for v := range s.m {
		values = append(values, v)
	}
	return json.Marshal(values)
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
