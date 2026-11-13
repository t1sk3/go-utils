package multimap

import (
	"fmt"
	"slices"
	"testing"
)

func TestZeroValue(t *testing.T) {
	var m Multimap[string, int]
	if !m.IsEmpty() || m.Len() != 0 || m.KeyLen() != 0 || m.Has("a") || m.Count("a") != 0 {
		t.Errorf("the zero value is not empty")
	}
	if m.Get("a") != nil || m.Remove("a") != nil || m.RemoveFunc("a", func(int) bool { return true }) != 0 {
		t.Errorf("reads of a missing key do not return nothing")
	}
	m.Clear()
	for range m.All() {
		t.Errorf("All() of the zero value produced a pair")
	}
	if m.String() != "Multimap[]" || m.Clone().Len() != 0 {
		t.Errorf("String() or Clone() of the zero value is wrong")
	}
	m.Put("a")
	if m.Has("a") || m.KeyLen() != 0 {
		t.Errorf("Put with no values added a key")
	}
}

func TestPutGetRemove(t *testing.T) {
	var m Multimap[string, int]
	m.Put("a", 1, 2)
	m.Put("b", 3)
	m.Put("a", 1)
	if got := m.Get("a"); !slices.Equal(got, []int{1, 2, 1}) {
		t.Errorf("Get(a) = %v", got)
	}
	if m.Len() != 4 || m.KeyLen() != 2 || m.Count("a") != 3 || !m.Has("b") || m.IsEmpty() {
		t.Errorf("counts are wrong: %v", &m)
	}
	if m.String() != "Multimap[a:[1 2 1] b:[3]]" {
		t.Errorf("String() = %q", &m)
	}

	// Appending to what Get returned must not write into the Multimap's storage.
	got := m.Get("b")
	_ = append(got, 99)
	m.Put("b", 4)
	if got := m.Get("b"); !slices.Equal(got, []int{3, 4}) {
		t.Errorf("Get(b) after appending to an earlier result = %v", got)
	}

	if n := m.RemoveFunc("a", func(v int) bool { return v == 1 }); n != 2 || m.Len() != 3 {
		t.Errorf("RemoveFunc removed %d, leaving %d", n, m.Len())
	}
	if n := m.RemoveFunc("a", func(v int) bool { return v == 2 }); n != 1 || m.Has("a") {
		t.Errorf("RemoveFunc of the last value removed %d and kept the key: %v", n, m.Has("a"))
	}
	if got := m.Remove("b"); !slices.Equal(got, []int{3, 4}) || m.Len() != 0 || m.KeyLen() != 0 {
		t.Errorf("Remove(b) = %v, leaving %v", got, &m)
	}
}

func TestIterationAndClone(t *testing.T) {
	var m Multimap[string, int]
	m.Put("a", 1, 2)
	m.Put("b", 3)

	var pairs []string
	for k, v := range m.All() {
		pairs = append(pairs, fmt.Sprintf("%v %v", k, v))
	}
	slices.Sort(pairs)
	if !slices.Equal(pairs, []string{"a 1", "a 2", "b 3"}) {
		t.Errorf("All() = %v", pairs)
	}
	if keys := slices.Sorted(m.Keys()); !slices.Equal(keys, []string{"a", "b"}) {
		t.Errorf("Keys() = %v", keys)
	}
	groups := map[string][]int{}
	for k, values := range m.Groups() {
		groups[k] = values
	}
	if len(groups) != 2 || !slices.Equal(groups["a"], []int{1, 2}) {
		t.Errorf("Groups() = %v", groups)
	}
	for range m.All() {
		break
	}
	for range m.Groups() {
		break
	}

	c := m.Clone()
	c.Put("a", 9)
	c.Remove("b")
	if m.Count("a") != 2 || !m.Has("b") || c.Len() != 3 {
		t.Errorf("Clone shares storage: %v and %v", &m, c)
	}
	m.Clear()
	if !m.IsEmpty() || m.KeyLen() != 0 || c.Len() != 3 {
		t.Errorf("Clear left %v, clone %v", &m, c)
	}
}
