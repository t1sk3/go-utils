package set

import (
	"encoding/json"
	"slices"
	"testing"
)

func sorted(s Set[int]) []int {
	return slices.Sorted(s.All())
}

func TestZeroValueIsUsable(t *testing.T) {
	var s Set[int]
	if !s.IsEmpty() || s.Len() != 0 || s.Has(1) {
		t.Errorf("zero Set is not empty")
	}
	s.Remove(1)
	s.Clear()
	s.Add(1, 2, 2)
	if s.Len() != 2 || !s.Has(1) || !s.Has(2) {
		t.Errorf("Add on zero Set = %v", s)
	}
}

func TestAddRemoveClear(t *testing.T) {
	s := Of(1, 2, 3)
	s.Remove(2, 9)
	if got := sorted(s); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("after Remove = %v", got)
	}
	s.Clear()
	if !s.IsEmpty() {
		t.Errorf("after Clear = %v", s)
	}
}

func TestCollect(t *testing.T) {
	s := Collect(slices.Values([]int{3, 1, 3, 2}))
	if got := sorted(s); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Collect = %v", got)
	}
}

func TestCloneDoesNotShareStorage(t *testing.T) {
	a := Of(1)
	b := a.Clone()
	b.Add(2)
	if a.Has(2) {
		t.Errorf("Clone shares storage with the original")
	}
	var zero Set[int]
	c := zero.Clone()
	c.Add(1)
	if zero.Has(1) {
		t.Errorf("Clone of zero Set shares storage")
	}
}

func TestSetAlgebra(t *testing.T) {
	a, b := Of(1, 2, 3), Of(2, 3, 4)
	tests := []struct {
		name string
		got  Set[int]
		want []int
	}{
		{"Union", a.Union(b), []int{1, 2, 3, 4}},
		{"Intersection", a.Intersection(b), []int{2, 3}},
		{"Difference", a.Difference(b), []int{1}},
		{"SymmetricDifference", a.SymmetricDifference(b), []int{1, 4}},
		{"Union with zero", a.Union(Set[int]{}), []int{1, 2, 3}},
		{"Intersection with zero", Set[int]{}.Intersection(a), []int{}},
	}
	for _, tt := range tests {
		if got := sorted(tt.got); !slices.Equal(got, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, got, tt.want)
		}
	}
	if !slices.Equal(sorted(a), []int{1, 2, 3}) {
		t.Errorf("operations modified their receiver: %v", a)
	}
}

func TestSubsetAndEqual(t *testing.T) {
	a, b := Of(1, 2), Of(1, 2, 3)
	if !a.IsSubsetOf(b) || b.IsSubsetOf(a) {
		t.Errorf("IsSubsetOf wrong")
	}
	if !(Set[int]{}).IsSubsetOf(a) {
		t.Errorf("empty set is not a subset")
	}
	if a.Equal(b) || !a.Equal(Of(2, 1)) || !(Set[int]{}).Equal(Of[int]()) {
		t.Errorf("Equal wrong")
	}
}

func TestString(t *testing.T) {
	if got := Of("b", "c", "a").String(); got != "Set[a b c]" {
		t.Errorf("String() = %q", got)
	}
	if got := (Set[int]{}).String(); got != "Set[]" {
		t.Errorf("String() of empty = %q", got)
	}
}

func TestJSON(t *testing.T) {
	data, err := json.Marshal(Of(1))
	if err != nil || string(data) != "[1]" {
		t.Fatalf("Marshal = %s, %v", data, err)
	}
	if data, _ := json.Marshal(Set[int]{}); string(data) != "[]" {
		t.Errorf("Marshal of zero Set = %s, want []", data)
	}

	var s Set[int]
	if err := json.Unmarshal([]byte("[3, 1, 3]"), &s); err != nil {
		t.Fatal(err)
	}
	if got := sorted(s); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("Unmarshal = %v", got)
	}
	if err := json.Unmarshal([]byte(`{"a":1}`), &s); err == nil {
		t.Errorf("Unmarshal of an object did not fail")
	}

	var wrapped struct{ Tags Set[string] }
	if err := json.Unmarshal([]byte(`{"Tags":["x","y"]}`), &wrapped); err != nil || !wrapped.Tags.Has("y") {
		t.Errorf("Unmarshal into struct field = %v, %v", wrapped.Tags, err)
	}
}

func TestAddAndRemoveCounts(t *testing.T) {
	var s Set[int]
	if got := s.Add(1, 2, 2); got != 2 {
		t.Errorf("Add new values = %d, want 2", got)
	}
	if got := s.Add(2, 3); got != 1 {
		t.Errorf("Add with an existing value = %d, want 1", got)
	}
	if got := s.Add(); got != 0 {
		t.Errorf("Add nothing = %d, want 0", got)
	}
	if got := s.Remove(1, 1, 9); got != 1 {
		t.Errorf("Remove = %d, want 1", got)
	}
	var zero Set[int]
	if got := zero.Remove(1); got != 0 {
		t.Errorf("Remove on the zero value = %d, want 0", got)
	}
}

func TestRemoveIf(t *testing.T) {
	s := Of(1, 2, 3, 4)
	if got := s.RemoveIf(func(v int) bool { return v%2 == 0 }); got != 2 {
		t.Errorf("RemoveIf = %d, want 2", got)
	}
	if got := sorted(s); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("left %v", got)
	}
	var zero Set[int]
	if got := zero.RemoveIf(func(int) bool { return true }); got != 0 {
		t.Errorf("RemoveIf on the zero value = %d, want 0", got)
	}
}

func TestRetainAll(t *testing.T) {
	s := Of(1, 2, 3)
	if got := s.RetainAll(Of(2, 3, 4)); got != 1 {
		t.Errorf("RetainAll = %d, want 1", got)
	}
	if got := sorted(s); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("left %v", got)
	}
	if got := s.RetainAll(Set[int]{}); got != 2 || !s.IsEmpty() {
		t.Errorf("RetainAll of the zero value = %d, left %v", got, s)
	}
}

func TestIsSupersetOf(t *testing.T) {
	s := Of(1, 2, 3)
	if !s.IsSupersetOf(Of(1, 2)) || !s.IsSupersetOf(s) || !s.IsSupersetOf(Set[int]{}) {
		t.Error("expected a superset")
	}
	if s.IsSupersetOf(Of(1, 4)) || (Set[int]{}).IsSupersetOf(s) {
		t.Error("unexpected superset")
	}
}

func TestIsDisjoint(t *testing.T) {
	if !Of(1, 2).IsDisjoint(Of(3, 4, 5)) || !Of(1).IsDisjoint(Set[int]{}) || !(Set[int]{}).IsDisjoint(Set[int]{}) {
		t.Error("expected disjoint sets")
	}
	if Of(1, 2).IsDisjoint(Of(2, 3, 4)) || Of(2, 3, 4).IsDisjoint(Of(1, 2)) {
		t.Error("unexpected disjoint sets")
	}
}
