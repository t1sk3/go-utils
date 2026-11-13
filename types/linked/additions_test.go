package linked

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
)

func TestMapJSON(t *testing.T) {
	m := NewMap[string, int]()
	m.Put("zebra", 1)
	m.Put("apple", 2)
	m.Put(`quo"te`, 3)
	data, err := json.Marshal(m)
	if err != nil || string(data) != `{"zebra":1,"apple":2,"quo\"te":3}` {
		t.Fatalf("Marshal = %s, %v", data, err)
	}
	if data, _ := json.Marshal(NewMap[string, int]()); string(data) != "{}" {
		t.Errorf("Marshal of an empty Map = %s", data)
	}

	back := NewMap[string, int]()
	back.Put("old", 0)
	if err := json.Unmarshal([]byte(` {"b":1, "a":2, "c":{"skip":0}} `), NewMap[string, any]()); err != nil {
		t.Errorf("Unmarshal with a nested value: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"b":1,"a":2,"b":3}`), back); err != nil {
		t.Fatal(err)
	}
	if got := slices.Collect(back.Keys()); !slices.Equal(got, []string{"b", "a"}) || back.Get("b").OrElse(0) != 3 || back.Has("old") {
		t.Errorf("Unmarshal = %v, want order b a with the last b and nothing older", back)
	}
	if err := json.Unmarshal([]byte("null"), back); err != nil || !back.IsEmpty() {
		t.Errorf("Unmarshal of null = %v, %v", back, err)
	}
	for _, bad := range []string{`[1]`, `{"a":"x"}`, `{"a":1`, `{"a"`, ``, `{"a":1]`} {
		if err := json.Unmarshal([]byte(bad), back); err == nil {
			t.Errorf("Unmarshal(%s) did not fail", bad)
		}
		if err := back.UnmarshalJSON([]byte(bad)); err == nil {
			t.Errorf("UnmarshalJSON(%s) did not fail", bad)
		}
	}

	var wrapped struct{ Fields *Map[string, bool] }
	if err := json.Unmarshal([]byte(`{"Fields":{"y":true,"x":false}}`), &wrapped); err != nil || wrapped.Fields.First().MustGet().First != "y" {
		t.Errorf("Unmarshal into a struct field = %v, %v", wrapped.Fields, err)
	}
}

func TestMapJSONKeys(t *testing.T) {
	ints := NewMap[int, string]()
	ints.Put(10, "a")
	ints.Put(-2, "b")
	data, err := json.Marshal(ints)
	if err != nil || string(data) != `{"10":"a","-2":"b"}` {
		t.Fatalf("Marshal with integer keys = %s, %v", data, err)
	}
	back := NewMap[int, string]()
	if err := json.Unmarshal(data, back); err != nil || !slices.Equal(slices.Collect(back.Keys()), []int{10, -2}) {
		t.Errorf("Unmarshal with integer keys = %v, %v", back, err)
	}
	if err := json.Unmarshal([]byte(`{"x":"a"}`), back); err == nil {
		t.Errorf("Unmarshal of a non-numeric key into an int did not fail")
	}

	type point struct{ X, Y int }
	structs := NewMap[point, int]()
	structs.Put(point{1, 2}, 1)
	if _, err := json.Marshal(structs); err == nil {
		t.Errorf("Marshal with struct keys did not fail")
	}
	floats := NewMap[float64, int]()
	floats.Put(math.NaN(), 1)
	if _, err := json.Marshal(floats); err == nil {
		t.Errorf("Marshal with a key JSON cannot hold did not fail")
	}
	values := NewMap[string, float64]()
	values.Put("a", math.Inf(1))
	if _, err := json.Marshal(values); err == nil {
		t.Errorf("Marshal with a value JSON cannot hold did not fail")
	}
}

func TestSetJSON(t *testing.T) {
	s := NewSet("c", "a", "b")
	data, err := json.Marshal(s)
	if err != nil || string(data) != `["c","a","b"]` {
		t.Fatalf("Marshal = %s, %v", data, err)
	}
	if data, _ := json.Marshal(NewSet[int]()); string(data) != "[]" {
		t.Errorf("Marshal of an empty Set = %s", data)
	}
	back := NewSet("old")
	if err := json.Unmarshal([]byte(`["y","x","y"]`), back); err != nil || !slices.Equal(slices.Collect(back.All()), []string{"y", "x"}) {
		t.Errorf("Unmarshal = %v, %v", back, err)
	}
	if err := json.Unmarshal([]byte(`{"a":1}`), back); err == nil {
		t.Errorf("Unmarshal of an object did not fail")
	}
}

func TestLRU(t *testing.T) {
	var evicted []string
	c := NewLRU[string, int](2).OnEvict(func(k string, v int) { evicted = append(evicted, k) })
	c.Put("a", 1)
	c.Put("b", 2)
	if v, ok := c.Get("a").Value(); !ok || v != 1 {
		t.Errorf("Get(a) = %v, %v", v, ok)
	}
	c.Put("c", 3) // b is now the least recently used
	if !slices.Equal(evicted, []string{"b"}) || c.Has("b") || c.Len() != 2 || c.Cap() != 2 {
		t.Errorf("evicted %v, want b; cache holds %d", evicted, c.Len())
	}
	// Peek and Has do not count as use, so a is still the one to go.
	c.Peek("a")
	c.Has("a")
	c.Put("d", 4)
	if !slices.Equal(evicted, []string{"b", "a"}) {
		t.Errorf("evicted %v, want b then a", evicted)
	}
	// Replacing a value counts as use and evicts nothing.
	c.Put("c", 30)
	c.Put("e", 5)
	if !slices.Equal(evicted, []string{"b", "a", "d"}) || c.Peek("c").OrElse(0) != 30 {
		t.Errorf("evicted %v, want b a d", evicted)
	}
	var order []string
	for k := range c.All() {
		order = append(order, k)
	}
	if !slices.Equal(order, []string{"c", "e"}) {
		t.Errorf("All() = %v, want least to most recently used", order)
	}
	if c.Get("zzz").IsPresent() || c.Peek("zzz").IsPresent() {
		t.Errorf("a missing key was found")
	}

	if v, ok := c.Remove("c").Value(); !ok || v != 30 || len(evicted) != 3 {
		t.Errorf("Remove(c) = %v, %v, or it called the eviction function", v, ok)
	}
	c.Resize(3)
	c.Put("f", 6)
	c.Put("g", 7)
	c.Resize(1)
	if c.Len() != 1 || !c.Has("g") || !slices.Equal(evicted[3:], []string{"e", "f"}) {
		t.Errorf("after Resize(1): %d entries, evicted %v", c.Len(), evicted)
	}
	c.Clear()
	if c.Len() != 0 || len(evicted) != 5 {
		t.Errorf("Clear left %d entries or called the eviction function", c.Len())
	}

	plain := NewLRU[int, int](1)
	plain.Put(1, 1)
	plain.Put(2, 2) // evicts with no function set
	if plain.Has(1) || !plain.Has(2) {
		t.Errorf("an LRU without an eviction function did not evict")
	}
	for name, f := range map[string]func(){
		"NewLRU(0)":  func() { NewLRU[int, int](0) },
		"Resize(-1)": func() { plain.Resize(-1) },
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
