package set

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestMarshalJSONIsDeterministic(t *testing.T) {
	s := Of("pear", "apple", "fig", "kiwi", "plum", "lime")
	const want = `["apple","fig","kiwi","lime","pear","plum"]`
	for range 20 {
		data, err := json.Marshal(s)
		if err != nil || string(data) != want {
			t.Fatalf("Marshal = %s, %v, want %s", data, err, want)
		}
	}
	// The order is that of the encoded bytes, not of the numbers.
	if data, _ := json.Marshal(Of(9, 10, 2)); string(data) != "[10,2,9]" {
		t.Errorf("Marshal of numbers = %s", data)
	}
	if _, err := json.Marshal(Of(math.Inf(1))); err == nil {
		t.Errorf("Marshal of a value JSON cannot hold did not fail")
	}
}

func TestRemoveIfRemovesNaN(t *testing.T) {
	var s Set[float64]
	s.Add(math.NaN(), math.NaN(), 1, 2)
	if s.Len() != 4 || s.Has(math.NaN()) || s.Remove(math.NaN()) != 0 {
		t.Fatalf("NaN is not a separate, unfindable value each time: %v", s)
	}
	if got := s.RemoveIf(func(v float64) bool { return v == 1 }); got != 1 || s.Len() != 3 {
		t.Errorf("RemoveIf(== 1) removed %d, leaving %d", got, s.Len())
	}
	if got := s.RemoveIf(math.IsNaN); got != 2 || s.Len() != 1 || !s.Has(2) {
		t.Errorf("RemoveIf(IsNaN) removed %d, leaving %v", got, s)
	}

	// Removing only some of the values that are not equal to themselves keeps the rest.
	type key struct {
		f  float64
		id int
	}
	k := Of(key{math.NaN(), 1}, key{math.NaN(), 2}, key{0, 3})
	if got := k.RemoveIf(func(v key) bool { return v.id == 1 }); got != 1 || k.Len() != 2 {
		t.Errorf("RemoveIf(id == 1) removed %d, leaving %d", got, k.Len())
	}
	for v := range k.All() {
		if v.id == 1 {
			t.Errorf("id 1 is still present")
		}
	}
}

func TestUnhashableValuePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("no panic for an unhashable value")
		}
	}()
	var s Set[any]
	s.Add([]int{1})
}

func TestAddSeqPopHasToSlice(t *testing.T) {
	var s Set[int]
	if got := s.AddSeq(slices.Values([]int{1, 2, 2, 3})); got != 3 || s.Len() != 3 {
		t.Errorf("AddSeq added %d, leaving %d", got, s.Len())
	}
	if got := s.AddSeq(slices.Values([]int{3, 4})); got != 1 {
		t.Errorf("AddSeq with one new value added %d", got)
	}
	if !s.HasAll(1, 4) || s.HasAll(1, 9) || !s.HasAll() || !s.HasAny(9, 2) || s.HasAny(8, 9) || s.HasAny() {
		t.Errorf("HasAll or HasAny is wrong for %v", s)
	}
	if got := s.ToSlice(); !slices.Equal(slices.Sorted(slices.Values(got)), []int{1, 2, 3, 4}) {
		t.Errorf("ToSlice() = %v", got)
	}
	if got := (Set[int]{}).ToSlice(); got == nil || len(got) != 0 {
		t.Errorf("ToSlice() of an empty Set = %#v", got)
	}

	var popped []int
	for v, ok := s.Pop().Value(); ok; v, ok = s.Pop().Value() {
		popped = append(popped, v)
	}
	slices.Sort(popped)
	if !slices.Equal(popped, []int{1, 2, 3, 4}) || !s.IsEmpty() {
		t.Errorf("Pop() until empty gave %v, leaving %v", popped, s)
	}
	var empty Set[int]
	if empty.Pop().IsPresent() {
		t.Errorf("Pop() of the zero value returned a value")
	}

	nan := Of(math.NaN(), math.NaN())
	for i := 0; !nan.IsEmpty(); i++ {
		if v := nan.Pop().OrElse(0); !math.IsNaN(v) || i > 2 {
			t.Fatalf("Pop() of NaN = %v on call %d", v, i)
		}
	}
}
