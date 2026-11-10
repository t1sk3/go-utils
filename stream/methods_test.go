package stream

import (
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/t1sk3/go-utils/types/pair"
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

func TestStreamMethodResults(t *testing.T) {
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

func TestSliceAndPairMethodsInferTheirTypes(t *testing.T) {
	// S and P are inferred as exactly []T and pair.Pair[...], so the results have the
	// same types the former functions returned and fit where those are expected.
	var chunks Stream[[]int] = Range(0, 5).Chunk(2)
	var windows Stream[[]int] = Range(0, 4).Window(2)
	var indexed Stream[pair.Pair[int, string]] = Of("a", "b").Enumerate()
	var zipped Stream[pair.Pair[int, string]] = Of(1, 2).Zip(Of("x", "y"))
	if got := chunks.Count() + windows.Count() + indexed.Count() + zipped.Count(); got != 3+3+2+2 {
		t.Errorf("counts = %d", got)
	}
	// ToSeq2 takes a Stream[pair.Pair[K, V]], so this only compiles if P was inferred.
	if got := maps.Collect(ToSeq2(Of("a", "b").Enumerate())); !maps.Equal(got, map[int]string{0: "a", 1: "b"}) {
		t.Errorf("ToSeq2(Enumerate) = %v", got)
	}
	// Explicit type arguments are accepted too.
	if got := Range(0, 4).Chunk[[]int](3).Count(); got != 2 {
		t.Errorf("Chunk[[]int] count = %d", got)
	}
}

func TestSliceAndPairMethodsChain(t *testing.T) {
	// Each call on the result needs a new instantiation, Stream[[][]int] and so on,
	// which is what the direct form Stream[[]T] could not declare.
	got := Range(0, 5).Window(2).Window(2).Map(func(w [][]int) int { return w[1][1] }).ToSlice()
	if want := []int{2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("Window.Window.Map = %v, want %v", got, want)
	}
	sums := Range(0, 7).Chunk(3).Map(func(c []int) int { return Sum(FromSlice(c)) }).ToSlice()
	if want := []int{3, 12, 6}; !slices.Equal(sums, want) {
		t.Errorf("Chunk.Map = %v, want %v", sums, want)
	}
	labels := Of("a", "b").Enumerate().Zip(Of(true, false)).Map(func(p pair.Pair[pair.Pair[int, string], bool]) string {
		return strconv.Itoa(p.First.First) + p.First.Second + strconv.FormatBool(p.Second)
	}).ToSlice()
	if want := []string{"0atrue", "1bfalse"}; !slices.Equal(labels, want) {
		t.Errorf("Enumerate.Zip.Map = %v, want %v", labels, want)
	}
}
