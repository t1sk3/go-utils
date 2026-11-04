package stream

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

type person struct {
	name string
	age  int
}

func personAge(p person) int     { return p.age }
func personName(p person) string { return p.name }

func TestComparators(t *testing.T) {
	alice, bob, carol, dave := person{"alice", 30}, person{"bob", 25}, person{"carol", 30}, person{"dave", 25}
	people := Of(carol, bob, alice, dave)
	byAge := Comparing(personAge)
	byName := Comparing(personName)

	tests := []struct {
		name string
		cmp  func(a, b person) int
		want []person
	}{
		{"Comparing is stable", byAge, []person{bob, dave, carol, alice}},
		{"Reversed keeps ties stable", Reversed(byAge), []person{carol, alice, bob, dave}},
		{"ThenComparing", ThenComparing(byAge, byName), []person{bob, dave, alice, carol}},
		{"ThenComparing Reversed", ThenComparing(Reversed(byAge), byName), []person{alice, carol, bob, dave}},
		{"Reversed ThenComparing", Reversed(ThenComparing(byAge, byName)), []person{carol, alice, dave, bob}},
	}
	for _, tt := range tests {
		if got := people.Sorted(tt.cmp).ToSlice(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}

	if got := people.Max(byAge).Get(); got != carol {
		t.Errorf("Max: got %v, want the first of the oldest", got)
	}
	if got := people.Min(Reversed(byAge)).Get(); got != carol {
		t.Errorf("Min of Reversed: got %v, want the first of the oldest", got)
	}
}

func TestThenComparing_OnlyCallsSecondOnTies(t *testing.T) {
	calls := 0
	second := func(a, b int) int { calls++; return 0 }
	c := ThenComparing(func(a, b int) int { return a - b }, second)

	c(1, 2)
	if calls != 0 {
		t.Errorf("second was called %d times for a non-tie", calls)
	}
	c(1, 1)
	if calls != 1 {
		t.Errorf("second was called %d times for a tie, want 1", calls)
	}
}

func TestRangeClosed(t *testing.T) {
	tests := []struct {
		name       string
		start, end int
		want       []int
	}{
		{"ascending", 1, 3, []int{1, 2, 3}},
		{"single", 2, 2, []int{2}},
		{"empty", 3, 2, []int{}},
		{"negative", -1, 1, []int{-1, 0, 1}},
	}
	for _, tt := range tests {
		s := RangeClosed(tt.start, tt.end)
		if got := s.ToSlice(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
		if s.size != len(tt.want) {
			t.Errorf("%s: size = %d, want %d", tt.name, s.size, len(tt.want))
		}
	}
}

func TestRangeClosed_TypeLimits(t *testing.T) {
	if got := RangeClosed[int8](125, math.MaxInt8).ToSlice(); !slices.Equal(got, []int8{125, 126, 127}) {
		t.Errorf("up to MaxInt8: got %v", got)
	}
	if got := RangeClosed[uint8](254, math.MaxUint8).ToSlice(); !slices.Equal(got, []uint8{254, 255}) {
		t.Errorf("up to MaxUint8: got %v", got)
	}
	if got := RangeClosed[uint8](0, 0).ToSlice(); !slices.Equal(got, []uint8{0}) {
		t.Errorf("unsigned single zero: got %v", got)
	}

	// The whole type: the difference overflows int8, so the size is unknown.
	full := RangeClosed[int8](math.MinInt8, math.MaxInt8)
	if got := full.Count(); got != 256 {
		t.Errorf("full int8 range: Count = %d, want 256", got)
	}
	if full.size != -1 {
		t.Errorf("full int8 range: size = %d, want -1", full.size)
	}

	// The size would overflow int, so it must be reported as unknown.
	if s := RangeClosed[int](math.MinInt, math.MaxInt); s.size != -1 {
		t.Errorf("full int range: size = %d, want -1", s.size)
	}
	if s := RangeClosed[int](0, math.MaxInt); s.size != -1 {
		t.Errorf("0..MaxInt: size = %d, want -1", s.size)
	}
	if s := RangeClosed[uint64](0, math.MaxUint64); s.size != -1 {
		t.Errorf("full uint64 range: size = %d, want -1", s.size)
	}
	if s := RangeClosed[int](1, math.MaxInt); s.size != math.MaxInt {
		t.Errorf("1..MaxInt: size = %d, want MaxInt", s.size)
	}
}

func TestRangeClosed_StopsEarly(t *testing.T) {
	if got := RangeClosed[int](0, math.MaxInt).Limit(2).ToSlice(); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("Limit: got %v", got)
	}
}

func TestAverage(t *testing.T) {
	if got := Average(Of(1, 2, 4)).Get(); got != 7.0/3.0 {
		t.Errorf("ints: got %v", got)
	}
	if got := Average(Of(1.5, 2.5)).Get(); got != 2 {
		t.Errorf("floats: got %v", got)
	}
	if got := Average(Range(0, 4)).Get(); got != 1.5 {
		t.Errorf("lazy stream: got %v", got)
	}
	if Average(Empty[int]()).IsPresent() {
		t.Error("empty stream: want an empty Optional")
	}

	// Would wrap around if the elements were summed as int8.
	if got := Average(Of[int8](100, 100)).Get(); got != 100 {
		t.Errorf("int8: got %v", got)
	}
	type celsius float64
	if got := Average(Of[celsius](10, 20)).Get(); got != 15 {
		t.Errorf("named type: got %v", got)
	}
}

func TestCountBy(t *testing.T) {
	got := CountBy(Of("a", "bb", "c", "dd", "eee"), func(s string) int { return len(s) })
	if want := map[int]int{1: 2, 2: 2, 3: 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := CountBy(Empty[string](), func(s string) int { return len(s) }); got == nil || len(got) != 0 {
		t.Errorf("empty stream: got %#v, want an empty non-nil map", got)
	}
}

func TestToMapMerge(t *testing.T) {
	id := func(s string) string { return s }
	first := func(s string) byte { return s[0] }
	got := ToMapMerge(Of("apple", "avocado", "banana", "apricot"), first, id,
		func(existing, next string) string { return existing + "," + next })
	want := map[byte]string{'a': "apple,avocado,apricot", 'b': "banana"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("merge order: got %v, want %v", got, want)
	}

	// merge must also run when the existing value is the zero value.
	calls := 0
	ToMapMerge(Of(0, 0), func(v int) int { return v }, func(v int) int { return v },
		func(a, b int) int { calls++; return a + b })
	if calls != 1 {
		t.Errorf("merge called %d times, want 1", calls)
	}

	if got := ToMapMerge(Empty[string](), id, id, func(a, _ string) string { return a }); got == nil || len(got) != 0 {
		t.Errorf("empty stream: got %#v, want an empty non-nil map", got)
	}
}

func TestBatchOne_NilFunctionPanics(t *testing.T) {
	cmpInts := func(a, b int) int { return a - b }
	self := func(v int) int { return v }
	for name, f := range map[string]func(){
		"Comparing":            func() { Comparing[int, int](nil) },
		"Reversed":             func() { Reversed[int](nil) },
		"ThenComparing first":  func() { ThenComparing(nil, cmpInts) },
		"ThenComparing second": func() { ThenComparing(cmpInts, nil) },
		"CountBy":              func() { CountBy[int, int](Of(1), nil) },
		"ToMapMerge key":       func() { ToMapMerge[int, int](Of(1), nil, self, cmpInts) },
		"ToMapMerge value":     func() { ToMapMerge(Of(1), self, nil, cmpInts) },
		"ToMapMerge merge":     func() { ToMapMerge(Of(1), self, self, nil) },
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
