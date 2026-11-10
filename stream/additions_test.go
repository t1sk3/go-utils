package stream

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/t1sk3/go-utils/types/optional"
)

func TestFromOptional(t *testing.T) {
	if got := FromOptional(optional.Of(5)).ToSlice(); !slices.Equal(got, []int{5}) {
		t.Errorf("present = %v", got)
	}
	if got := FromOptional(optional.Of(0)).Count(); got != 1 {
		t.Errorf("a present zero value gave %d elements", got)
	}
	if got := FromOptional(optional.Empty[int]()).Count(); got != 0 {
		t.Errorf("empty gave %d elements", got)
	}
}

func TestOfNullable(t *testing.T) {
	v := 5
	s := OfNullable(&v)
	v = 6 // The value was copied when the stream was built.
	if got := s.ToSlice(); !slices.Equal(got, []int{5}) {
		t.Errorf("non-nil = %v", got)
	}
	if got := OfNullable[int](nil).Count(); got != 0 {
		t.Errorf("nil gave %d elements", got)
	}
}

func TestIterateWhile(t *testing.T) {
	below := func(limit int) func(int) bool { return func(i int) bool { return i < limit } }
	double := func(i int) int { return i * 2 }
	tests := []struct {
		name string
		s    Stream[int]
		want []int
	}{
		{"stops at the predicate", IterateWhile(1, below(20), double), []int{1, 2, 4, 8, 16}},
		{"seed fails", IterateWhile(100, below(20), double), []int{}},
		{"limit stops first", IterateWhile(1, below(20), double).Limit(2), []int{1, 2}},
	}
	for _, tt := range tests {
		if got := tt.s.ToSlice(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIterateWhileIsReusableAndDoesNotOverstep(t *testing.T) {
	nextCalls := 0
	s := IterateWhile(0, func(i int) bool { return i < 2 }, func(i int) int { nextCalls++; return i + 1 })
	if got := s.FindFirst().MustGet(); got != 0 || nextCalls != 0 {
		t.Errorf("FindFirst = %d after %d calls to next", got, nextCalls)
	}
	if a, b := s.Count(), s.Count(); a != 2 || b != 2 {
		t.Errorf("Count = %d then %d, want 2 each", a, b)
	}
}

func TestSummarize(t *testing.T) {
	got := Summarize(Of(3, 1, 2, 6))
	want := Summary[int]{Count: 4, Sum: 12, Min: 1, Max: 6, Average: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if got := Summarize(Empty[int]()); got != (Summary[int]{}) {
		t.Errorf("got %+v", got)
	}
}

func TestSummarizeOneElementFromSeq(t *testing.T) {
	got := Summarize(FromSeq(slices.Values([]float64{-2.5})))
	want := Summary[float64]{Count: 1, Sum: -2.5, Min: -2.5, Max: -2.5, Average: -2.5}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSummarizeAverageDoesNotWrap(t *testing.T) {
	got := Summarize(Of[uint8](200, 200))
	if got.Average != 200 {
		t.Errorf("Average = %v, want 200", got.Average)
	}
	if got.Sum != 144 {
		t.Errorf("Sum = %d, want the wrapped value 144", got.Sum)
	}
}

func TestSummarizeMatchesMinAndMaxWithNaN(t *testing.T) {
	values := []float64{1, math.NaN(), 2}
	got := Summarize(FromSlice(values))
	if !math.IsNaN(got.Min) || !math.IsNaN(Min(FromSlice(values)).MustGet()) {
		t.Errorf("Min = %v, want NaN to match the Min function", got.Min)
	}
	if got.Max != 2 || Max(FromSlice(values)).MustGet() != 2 {
		t.Errorf("Max = %v, want 2 to match the Max function", got.Max)
	}
}

func TestIdentity(t *testing.T) {
	got := Of("a", "bb").ToMap(Identity, func(s string) int { return len(s) })
	if want := map[string]int{"a": 1, "bb": 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNilComparators(t *testing.T) {
	one, two := 1, 2
	show := func(ps []*int) string {
		out := ""
		for _, p := range ps {
			if p == nil {
				out += "nil "
			} else {
				out += fmt.Sprint(*p, " ")
			}
		}
		return out
	}
	values := []*int{&two, nil, &one, nil}
	if got := show(FromSlice(values).Sorted(NilsFirst(cmp.Compare[int])).ToSlice()); got != "nil nil 1 2 " {
		t.Errorf("NilsFirst = %s", got)
	}
	if got := show(FromSlice(values).Sorted(NilsLast(cmp.Compare[int])).ToSlice()); got != "1 2 nil nil " {
		t.Errorf("NilsLast = %s", got)
	}
	if got := show(FromSlice(values).Sorted(Reversed(NilsLast(cmp.Compare[int]))).ToSlice()); got != "nil nil 2 1 " {
		t.Errorf("Reversed NilsLast = %s", got)
	}
}

func TestAdditionsRejectNilFunctions(t *testing.T) {
	inc := func(i int) int { return i + 1 }
	always := func(int) bool { return true }
	for name, f := range map[string]func(){
		"IterateWhile hasNext": func() { IterateWhile(0, nil, inc) },
		"IterateWhile next":    func() { IterateWhile(0, always, nil) },
		"NilsFirst":            func() { NilsFirst[int](nil) },
		"NilsLast":             func() { NilsLast[int](nil) },
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
