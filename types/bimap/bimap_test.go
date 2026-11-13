package bimap

import (
	"maps"
	"testing"
)

func TestZeroValue(t *testing.T) {
	var m BiMap[string, int]
	if !m.IsEmpty() || m.Len() != 0 || m.Has("a") || m.HasValue(1) {
		t.Errorf("the zero value is not empty")
	}
	if m.Get("a").IsPresent() || m.GetKey(1).IsPresent() || m.Remove("a").IsPresent() || m.RemoveValue(1).IsPresent() {
		t.Errorf("reads of the zero value found something")
	}
	m.Clear()
	if m.String() != "BiMap[]" || m.Clone().Len() != 0 || m.Inverse().Len() != 0 {
		t.Errorf("String(), Clone() or Inverse() of the zero value is wrong")
	}
	m.Clone().Put("a", 1)
	m.Inverse().Put(1, "a")
}

func TestBothDirections(t *testing.T) {
	var m BiMap[string, int]
	m.Put("a", 1)
	m.Put("b", 2)
	if v, ok := m.Get("a").Value(); !ok || v != 1 {
		t.Errorf("Get(a) = %v, %v", v, ok)
	}
	if k, ok := m.GetKey(2).Value(); !ok || k != "b" {
		t.Errorf("GetKey(2) = %v, %v", k, ok)
	}
	if !m.Has("a") || !m.HasValue(2) || m.Has("c") || m.HasValue(3) || m.Len() != 2 {
		t.Errorf("membership is wrong: %v", &m)
	}
	if m.String() != "BiMap[a:1 b:2]" {
		t.Errorf("String() = %q", &m)
	}
	if got := maps.Collect(m.All()); !maps.Equal(got, map[string]int{"a": 1, "b": 2}) {
		t.Errorf("All() = %v", got)
	}
}

// check verifies that the two directions describe the same pairs.
func check(t *testing.T, m *BiMap[string, int]) {
	t.Helper()
	if len(m.forward) != len(m.reverse) {
		t.Fatalf("%d keys but %d values", len(m.forward), len(m.reverse))
	}
	for k, v := range m.forward {
		if m.reverse[v] != k {
			t.Fatalf("%v maps to %v, which maps back to %v", k, v, m.reverse[v])
		}
	}
}

func TestPutKeepsOneToOne(t *testing.T) {
	var m BiMap[string, int]
	m.Put("a", 1)
	m.Put("b", 2)

	m.Put("a", 3) // replaces a's value; 1 is free again
	check(t, &m)
	if m.HasValue(1) || m.Len() != 2 {
		t.Errorf("replacing a key's value left the old value: %v", &m)
	}
	m.Put("c", 2) // takes b's value; b goes
	check(t, &m)
	if m.Has("b") || m.Len() != 2 {
		t.Errorf("reusing a value left its old key: %v", &m)
	}
	m.Put("a", 2) // collides on both sides: a:3 and c:2 both go
	check(t, &m)
	if m.Len() != 1 || m.Get("a").OrElse(0) != 2 {
		t.Errorf("a Put that collides on both sides left %v", &m)
	}
	m.Put("a", 2) // the same pair again
	check(t, &m)
	if m.Len() != 1 {
		t.Errorf("repeating a pair changed the size: %v", &m)
	}
}

func TestRemoveCloneInverse(t *testing.T) {
	var m BiMap[string, int]
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	inv := m.Inverse()
	if k, ok := inv.Get(2).Value(); !ok || k != "b" || inv.Len() != 3 {
		t.Errorf("Inverse() = %v", inv)
	}
	c := m.Clone()

	if v, ok := m.Remove("a").Value(); !ok || v != 1 || m.HasValue(1) {
		t.Errorf("Remove(a) = %v, %v", v, ok)
	}
	if k, ok := m.RemoveValue(2).Value(); !ok || k != "b" || m.Has("b") {
		t.Errorf("RemoveValue(2) = %v, %v", k, ok)
	}
	check(t, &m)
	if m.Len() != 1 || c.Len() != 3 || inv.Len() != 3 {
		t.Errorf("Clone or Inverse shares storage: %v, %v, %v", &m, c, inv)
	}
	m.Clear()
	if !m.IsEmpty() || m.HasValue(3) {
		t.Errorf("Clear left %v", &m)
	}
}
