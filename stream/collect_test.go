package stream

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/t1sk3/go-utils/types/pair"
)

func TestZip(t *testing.T) {
	want := []pair.Pair[int, string]{pair.Of(1, "a"), pair.Of(2, "b")}
	tests := []struct {
		name string
		got  Stream[pair.Pair[int, string]]
	}{
		{"slice b, a shorter", Zip(Of(1, 2), Of("a", "b", "c"))},
		{"slice b, b shorter", Zip(Range(1, 100), Of("a", "b"))},
		{"lazy b", Zip(Of(1, 2, 3), Of("a", "b").Filter(func(string) bool { return true }))},
		{"infinite a", Zip(Iterate(1, func(i int) int { return i + 1 }), Of("a", "b"))},
	}
	for _, tt := range tests {
		if got := tt.got.ToSlice(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Zip = %v, want %v", tt.name, got, want)
		}
	}
	if n := Zip(Of(1, 2), Empty[string]()).Count(); n != 0 {
		t.Errorf("Zip with empty = %d elements", n)
	}
	if got := Zip(Of(1, 2, 3), Of("a", "b")).size; got != 2 {
		t.Errorf("Zip size hint = %d, want 2", got)
	}
}

func TestZip_StopsPullingWhenBothSidesAreDone(t *testing.T) {
	pulledA, pulledB := 0, 0
	a := Range(0, 10).Peek(func(int) { pulledA++ })
	b := Range(0, 10).Peek(func(int) { pulledB++ })
	Zip(a, b).Limit(3).ToSlice()
	if pulledA != 3 || pulledB != 3 {
		t.Errorf("pulled %d from a and %d from b, want 3 and 3", pulledA, pulledB)
	}

	pulledA = 0
	Zip(a, Of(1, 2)).ToSlice()
	if pulledA != 2 {
		t.Errorf("pulled %d from a when zipped with a 2-element slice, want 2", pulledA)
	}
}

func TestEnumerate(t *testing.T) {
	got := Enumerate(Of("a", "b", "c").Skip(1)).ToSlice()
	want := []pair.Pair[int, string]{pair.Of(0, "b"), pair.Of(1, "c")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Enumerate = %v, want %v", got, want)
	}
	s := Enumerate(Of(1, 2))
	if first, second := s.ToSlice(), s.ToSlice(); !reflect.DeepEqual(first, second) {
		t.Errorf("Enumerate is not reusable: %v then %v", first, second)
	}
}

func TestSeq2RoundTrip(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	if got := maps.Collect(ToSeq2(FromMap(m))); !maps.Equal(got, m) {
		t.Errorf("FromMap -> ToSeq2 = %v, want %v", got, m)
	}
	if got := maps.Collect(ToSeq2(FromSeq2(maps.All(m)))); !maps.Equal(got, m) {
		t.Errorf("FromSeq2 -> ToSeq2 = %v, want %v", got, m)
	}
	if n := FromMap(m).size; n != 3 {
		t.Errorf("FromMap size hint = %d, want 3", n)
	}
	if n := FromSeq2[int, int](nil).Count(); n != 0 {
		t.Errorf("FromSeq2(nil) has %d elements", n)
	}

	var keys []int
	for i, v := range ToSeq2(Enumerate(Of("x", "y", "z"))) {
		if v == "z" {
			break
		}
		keys = append(keys, i)
	}
	if !slices.Equal(keys, []int{0, 1}) {
		t.Errorf("range over ToSeq2 with break = %v", keys)
	}
}

func TestToSet(t *testing.T) {
	s := ToSet(Of(3, 1, 3, 2))
	if got := slices.Sorted(s.All()); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("ToSet = %v", got)
	}
}

func TestChunk(t *testing.T) {
	tests := []struct {
		name string
		got  Stream[[]int]
		want [][]int
	}{
		{"uneven", Chunk(Range(0, 5), 2), [][]int{{0, 1}, {2, 3}, {4}}},
		{"even", Chunk(Range(0, 4), 2), [][]int{{0, 1}, {2, 3}}},
		{"larger than stream", Chunk(Of(1, 2), 10), [][]int{{1, 2}}},
		{"empty", Chunk(Empty[int](), 3), [][]int{}},
		{"infinite", Chunk(Iterate(0, func(i int) int { return i + 1 }), 2).Limit(2), [][]int{{0, 1}, {2, 3}}},
	}
	for _, tt := range tests {
		if got := tt.got.ToSlice(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Chunk = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := Chunk(Range(0, 5), 2).size; got != 3 {
		t.Errorf("Chunk size hint = %d, want 3", got)
	}
}

func TestChunkAndWindow_ReturnIndependentSlices(t *testing.T) {
	src := []int{1, 2, 3, 4}
	chunks := Chunk(FromSlice(src), 2).ToSlice()
	chunks[0][0] = 99
	chunks[0] = append(chunks[0], 100)
	if src[0] != 1 || chunks[1][0] != 3 {
		t.Errorf("chunks share memory: src=%v chunks=%v", src, chunks)
	}

	windows := Window(FromSlice(src), 2).ToSlice()
	windows[0][1] = 99
	if windows[1][0] != 2 || src[1] != 2 {
		t.Errorf("windows share memory: src=%v windows=%v", src, windows)
	}
}

func TestWindow(t *testing.T) {
	tests := []struct {
		name string
		got  Stream[[]int]
		want [][]int
	}{
		{"pairs", Window(Range(0, 4), 2), [][]int{{0, 1}, {1, 2}, {2, 3}}},
		{"size 1", Window(Of(5, 6), 1), [][]int{{5}, {6}}},
		{"exact", Window(Of(1, 2, 3), 3), [][]int{{1, 2, 3}}},
		{"too short", Window(Of(1, 2), 3), [][]int{}},
		{"infinite", Window(Iterate(0, func(i int) int { return i + 1 }), 3).Limit(2), [][]int{{0, 1, 2}, {1, 2, 3}}},
	}
	for _, tt := range tests {
		if got := tt.got.ToSlice(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Window = %v, want %v", tt.name, got, tt.want)
		}
	}
	if got := Window(Range(0, 4), 2).size; got != 3 {
		t.Errorf("Window size hint = %d, want 3", got)
	}
}

func TestScan(t *testing.T) {
	sum := func(acc, v int) int { return acc + v }
	if got := Scan(Of(1, 2, 3), 0, sum).ToSlice(); !slices.Equal(got, []int{1, 3, 6}) {
		t.Errorf("Scan = %v", got)
	}
	if got := Scan(Empty[int](), 10, sum).ToSlice(); len(got) != 0 {
		t.Errorf("Scan of empty = %v", got)
	}
	s := Scan(Iterate(1, func(i int) int { return i + 1 }), 0, sum).Limit(4)
	if first, second := s.ToSlice(), s.ToSlice(); !slices.Equal(first, []int{1, 3, 6, 10}) || !slices.Equal(first, second) {
		t.Errorf("Scan over infinite stream = %v then %v, want [1 3 6 10] twice", first, second)
	}
}

func TestPartition(t *testing.T) {
	even, odd := Range(0, 6).Partition(func(i int) bool { return i%2 == 0 })
	if !slices.Equal(even, []int{0, 2, 4}) || !slices.Equal(odd, []int{1, 3, 5}) {
		t.Errorf("Partition = %v, %v", even, odd)
	}
	matched, rest := Empty[int]().Partition(func(int) bool { return true })
	if matched == nil || rest == nil {
		t.Errorf("Partition of empty returned nil slices")
	}
}

func TestNewOperations_NilFunctionPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"Scan":      func() { Scan[int, int](Of(1), 0, nil) },
		"Partition": func() { Of(1).Partition(nil) },
		"Chunk(0)":  func() { Chunk(Of(1), 0) },
		"Window(0)": func() { Window(Of(1), -1) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			f()
		}()
	}
}
