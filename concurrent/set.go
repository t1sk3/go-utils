package concurrent

import (
	"iter"
	"maps"
	"slices"
	"sync"
)

// Set is a set that is safe for concurrent use. The zero value is empty and ready to
// use. A Set must not be copied after first use.
//
// Add reports whether the value was new, atomically, which makes it the way to let
// exactly one goroutine claim something:
//
//	if seen.Add(url) {
//		crawl(url) // no other goroutine crawls this URL
//	}
type Set[T comparable] struct {
	mu sync.RWMutex
	m  map[T]struct{}
}

// Add adds v and reports whether it was not already present.
func (s *Set[T]) Add(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[v]; ok {
		return false
	}
	if s.m == nil {
		s.m = make(map[T]struct{})
	}
	s.m[v] = struct{}{}
	return true
}

// Remove removes v and reports whether it was present.
func (s *Set[T]) Remove(v T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[v]
	delete(s.m, v)
	return ok
}

// Has reports whether v is present.
func (s *Set[T]) Has(v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[v]
	return ok
}

// Len returns the number of values in s.
func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// Clear removes every value from s.
func (s *Set[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.m)
}

// Snapshot returns a copy of the values as a slice that the caller owns, in no
// particular order. The result is never nil.
func (s *Set[T]) Snapshot() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.AppendSeq(make([]T, 0, len(s.m)), maps.Keys(s.m))
}

// All returns an iterator over a snapshot of the values, in no particular order. The
// Set is not locked during the loop, so its body may call any method of s, and changes
// made after the snapshot was taken are not seen.
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.Snapshot() {
			if !yield(v) {
				return
			}
		}
	}
}
