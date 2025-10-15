package utils

import (
	"reflect"
	"testing"
)

func TestFromSliceAndToSlice(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})
	result := s.ToSlice()
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("FromSlice/ToSlice failed: got %v, want %v", result, expected)
	}
}

func TestFilter(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	filtered := s.Filter(func(i int) bool { return i%2 == 0 }).ToSlice()
	expected := []int{2, 4}
	if !reflect.DeepEqual(filtered, expected) {
		t.Errorf("Filter failed: got %v, want %v", filtered, expected)
	}
}

func TestMap(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})
	mapped := s.Map(func(i int) any { return i * 2 }).ToSlice()
	expected := []any{2, 4, 6}
	if !reflect.DeepEqual(mapped, expected) {
		t.Errorf("Map failed: got %v, want %v", mapped, expected)
	}
}

func TestForEach(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})
	sum := 0
	s.ForEach(func(i int) { sum += i })
	if sum != 6 {
		t.Errorf("ForEach failed: got sum %d, want 6", sum)
	}
}

func TestFindFirst(t *testing.T) {
	s := FromSlice([]int{10, 20, 30})
	opt := s.FindFirst()
	val, ok := opt.Get(), opt.IsPresent()
	if !ok || val != 10 {
		t.Errorf("FindFirst failed: got %v, want 10", val)
	}
	empty := FromSlice([]int{}).FindFirst()
	if empty.IsPresent() {
		t.Errorf("FindFirst on empty slice should be empty")
	}
}

func TestFindAny(t *testing.T) {
	s := FromSlice([]int{42})
	opt := s.FindAny()
	val, ok := opt.Get(), opt.IsPresent()
	if !ok || val != 42 {
		t.Errorf("FindAny failed: got %v, want 42", val)
	}
}

func TestAllMatch(t *testing.T) {
	s := FromSlice([]int{2, 4, 6})
	if !s.AllMatch(func(i int) bool { return i%2 == 0 }) {
		t.Errorf("AllMatch failed: expected true")
	}
	if s.AllMatch(func(i int) bool { return i > 4 }) {
		t.Errorf("AllMatch failed: expected false")
	}
}

func TestAnyMatch(t *testing.T) {
	s := FromSlice([]int{1, 3, 5, 8})
	if !s.AnyMatch(func(i int) bool { return i%2 == 0 }) {
		t.Errorf("AnyMatch failed: expected true")
	}
	if s.AnyMatch(func(i int) bool { return i > 10 }) {
		t.Errorf("AnyMatch failed: expected false")
	}
}

func TestNoneMatch(t *testing.T) {
	s := FromSlice([]int{1, 3, 5})
	if !s.NoneMatch(func(i int) bool { return i%2 == 0 }) {
		t.Errorf("NoneMatch failed: expected true")
	}
	if s.NoneMatch(func(i int) bool { return i > 2 }) {
		t.Errorf("NoneMatch failed: expected false")
	}
}

func TestDistinct(t *testing.T) {
	s := FromSlice([]int{1, 2, 2, 3, 1, 4})
	distinct := s.Distinct().ToSlice()
	expected := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(distinct, expected) {
		t.Errorf("Distinct failed: got %v, want %v", distinct, expected)
	}
}

func TestSorted(t *testing.T) {
	s := FromSlice([]int{3, 1, 2})
	sorted := s.Sorted(func(a, b int) int { return a - b }).ToSlice()
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(sorted, expected) {
		t.Errorf("Sorted failed: got %v, want %v", sorted, expected)
	}
}

func TestCount(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4})
	if s.Count() != 4 {
		t.Errorf("Count failed: got %d, want 4", s.Count())
	}
}

func TestMin(t *testing.T) {
	s := FromSlice([]int{5, 2, 8, 1})
	min := s.Min(func(a, b int) int { return a - b })
	val, ok := min.Get(), min.IsPresent()
	if !ok || val != 1 {
		t.Errorf("Min failed: got %v, want 1", val)
	}
}

func TestMax(t *testing.T) {
	s := FromSlice([]int{5, 2, 8, 1})
	max := s.Max(func(a, b int) int { return a - b })
	val, ok := max.Get(), max.IsPresent()
	if !ok || val != 8 {
		t.Errorf("Max failed: got %v, want 8", val)
	}
}

func TestLimit(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	limited := s.Limit(3).ToSlice()
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(limited, expected) {
		t.Errorf("Limit failed: got %v, want %v", limited, expected)
	}
}

func TestSkip(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	skipped := s.Skip(2).ToSlice()
	expected := []int{3, 4, 5}
	if !reflect.DeepEqual(skipped, expected) {
		t.Errorf("Skip failed: got %v, want %v", skipped, expected)
	}
}

func TestReduce(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4})
	reduced := s.Reduce(func(a, b int) int { return a + b })
	val, ok := reduced.Get(), reduced.IsPresent()
	if !ok || val != 10 {
		t.Errorf("Reduce failed: got %v, want 10", val)
	}
}

func TestEmptyReduce(t *testing.T) {
	s := FromSlice([]int{})
	reduced := s.Reduce(func(a, b int) int { return a + b })
	if reduced.IsPresent() {
		t.Errorf("Reduce on empty slice should be empty")
	}
}

func TestFilterMapReduce(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5, 6})
	result := s.
		Filter(func(i int) bool { return i%2 == 0 }).
		Map(func(i int) any { return int64(i) * 10 }).
		Reduce(func(a, b any) any { return a.(int64) + b.(int64) })
	if result.Get() != int64(120) {
		t.Errorf("Combined Filter, Map, Reduce failed: got %d, want 120", result.Get())
	}
}
