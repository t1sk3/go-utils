package stream

import (
	"maps"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

func TestMapMethodChangesType(t *testing.T) {
	got := Range(1, 4).Map(strconv.Itoa).Map(func(s string) string { return s + "!" }).ToSlice()
	if want := []string{"1!", "2!", "3!"}; !slices.Equal(got, want) {
		t.Errorf("Map chain = %v, want %v", got, want)
	}
	// Same-type calls keep working without any change at the call site.
	if got := Of(1, 2).Map(func(i int) int { return i * 10 }).ToSlice(); !slices.Equal(got, []int{10, 20}) {
		t.Errorf("same-type Map = %v", got)
	}
	// Map keeps the known size, like the Map function.
	if got := Range(0, 5).Map(strconv.Itoa).size; got != 5 {
		t.Errorf("size after Map = %d, want 5", got)
	}
}

func TestParallelMapMethodChangesType(t *testing.T) {
	got := Range(0, 50).Parallel(4).
		Map(strconv.Itoa).
		Filter(func(s string) bool { return len(s) == 1 }).
		Map(func(s string) int { n, _ := strconv.Atoi(s); return n * n }).
		ToSlice()
	want := []int{0, 1, 4, 9, 16, 25, 36, 49, 64, 81}
	if !slices.Equal(got, want) {
		t.Errorf("parallel Map chain = %v, want %v", got, want)
	}
}

func TestStreamMethodsMatchFunctions(t *testing.T) {
	words := Of("apple", "avocado", "banana", "blueberry", "cherry", "apple")
	first := func(w string) byte { return w[0] }
	length := func(w string) int { return len(w) }
	add := func(a, b int) int { return a + b }
	sumLen := func(n int, w string) int { return n + len(w) }

	if got, want := words.FlatMap(func(w string) Stream[byte] { return Of([]byte(w)[:2]...) }).ToSlice(),
		FlatMap(words, func(w string) Stream[byte] { return Of([]byte(w)[:2]...) }).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("FlatMap = %q, want %q", got, want)
	}
	if got, want := words.DistinctBy(first).ToSlice(), DistinctBy(words, first).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("DistinctBy = %v, want %v", got, want)
	}
	if got, want := words.Fold(0, sumLen), Fold(words, 0, sumLen); got != want {
		t.Errorf("Fold = %d, want %d", got, want)
	}
	if got, want := words.Scan(0, sumLen).ToSlice(), Scan(words, 0, sumLen).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("Scan = %v, want %v", got, want)
	}
	if got, want := words.GroupBy(first), GroupBy(words, first); !reflect.DeepEqual(got, want) {
		t.Errorf("GroupBy = %v, want %v", got, want)
	}
	if got, want := words.CountBy(first), CountBy(words, first); !maps.Equal(got, want) {
		t.Errorf("CountBy = %v, want %v", got, want)
	}
	if got, want := words.ToMap(Identity, length), ToMap(words, Identity, length); !maps.Equal(got, want) {
		t.Errorf("ToMap = %v, want %v", got, want)
	}
	if got, want := words.ToMapMerge(first, length, add), ToMapMerge(words, first, length, add); !maps.Equal(got, want) {
		t.Errorf("ToMapMerge = %v, want %v", got, want)
	}
	if got, want := words.Collect(Joining[string]("+")), Collect(words, Joining[string]("+")); got != want {
		t.Errorf("Collect = %q, want %q", got, want)
	}
	if got, want := words.MapConcurrent(3, length).ToSlice(), MapConcurrent(words, 3, length).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("MapConcurrent = %v, want %v", got, want)
	}
}

func TestStreamMethodResults(t *testing.T) {
	// Spot-check actual values, so the comparison with the functions above cannot pass
	// just because both forms are wrong in the same way.
	s := Of(3, 1, 4, 1, 5, 9, 2, 6)
	if got := s.DistinctBy(func(i int) int { return i % 3 }).ToSlice(); !slices.Equal(got, []int{3, 1, 5}) {
		t.Errorf("DistinctBy = %v", got)
	}
	if got := s.Fold("", func(acc string, i int) string { return acc + strconv.Itoa(i) }); got != "31415926" {
		t.Errorf("Fold = %q", got)
	}
	if got := s.Limit(4).Scan(0, func(a, b int) int { return a + b }).ToSlice(); !slices.Equal(got, []int{3, 4, 8, 9}) {
		t.Errorf("Scan = %v", got)
	}
	if got := s.CountBy(IsEven[int]); !maps.Equal(got, map[bool]int{false: 5, true: 3}) {
		t.Errorf("CountBy = %v", got)
	}
	if got := s.FlatMap(func(i int) Stream[int] { return Range(0, i) }).Limit(5).ToSlice(); !slices.Equal(got, []int{0, 1, 2, 0, 0}) {
		t.Errorf("FlatMap = %v", got)
	}
	if got := s.Collect(Counting[int]()); got != 8 {
		t.Errorf("Collect(Counting) = %d", got)
	}
}

func TestStreamMethodsWorkOnInfiniteStreams(t *testing.T) {
	naturals := Iterate(1, func(i int) int { return i + 1 })
	if got := naturals.Scan(0, func(a, b int) int { return a + b }).Limit(4).ToSlice(); !slices.Equal(got, []int{1, 3, 6, 10}) {
		t.Errorf("Scan = %v", got)
	}
	if got := naturals.Map(strconv.Itoa).Limit(3).ToSlice(); !slices.Equal(got, []string{"1", "2", "3"}) {
		t.Errorf("Map = %v", got)
	}
	if got := naturals.MapConcurrent(4, func(i int) int { return -i }).Limit(3).ToSlice(); !slices.Equal(got, []int{-1, -2, -3}) {
		t.Errorf("MapConcurrent = %v", got)
	}
}

func TestStreamMethodsRejectNilFunctions(t *testing.T) {
	s := Of(1, 2)
	inc := func(i int) int { return i + 1 }
	for name, f := range map[string]func(){
		"Map":              func() { s.Map[int](nil) },
		"FlatMap":          func() { s.FlatMap[int](nil) },
		"DistinctBy":       func() { s.DistinctBy[int](nil) },
		"Fold":             func() { s.Fold(0, nil) },
		"Scan":             func() { s.Scan(0, nil) },
		"GroupBy":          func() { s.GroupBy[int](nil) },
		"CountBy":          func() { s.CountBy[int](nil) },
		"ToMap key":        func() { s.ToMap[int](nil, inc) },
		"ToMap value":      func() { s.ToMap(inc, (func(int) int)(nil)) },
		"ToMapMerge merge": func() { s.ToMapMerge(inc, inc, nil) },
		"Collect":          func() { s.Collect(Collector[int, int]{}) },
		"MapConcurrent":    func() { s.MapConcurrent[int](1, nil) },
		"MapConcurrent n":  func() { s.MapConcurrent(0, inc) },
		"Parallel Map":     func() { s.Parallel(1).Map[string](nil) },
	} {
		t.Run(name, func(t *testing.T) { mustPanic(t, f) })
	}
}
