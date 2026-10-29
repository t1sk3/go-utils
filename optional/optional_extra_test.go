package optional

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestZeroValueIsEmpty(t *testing.T) {
	var o Optional[string]
	if o.IsPresent() || !o.IsEmpty() {
		t.Errorf("zero Optional is not empty")
	}
}

func TestOfOk(t *testing.T) {
	m := map[string]int{"a": 1}
	v, ok := m["a"]
	if got := OfOk(v, ok); got.Get() != 1 || !got.IsPresent() {
		t.Errorf("OfOk(present) = %v", got)
	}
	v, ok = m["b"]
	if OfOk(v, ok).IsPresent() {
		t.Errorf("OfOk(missing) is present")
	}
}

func TestAccessors(t *testing.T) {
	full, empty := Of(5), Empty[int]()

	if v, ok := full.Value(); v != 5 || !ok {
		t.Errorf("Value() = %v, %v", v, ok)
	}
	if _, ok := empty.Value(); ok {
		t.Errorf("Value() on empty reported ok")
	}
	if full.OrElse(9) != 5 || empty.OrElse(9) != 9 {
		t.Errorf("OrElse wrong")
	}
	called := false
	full.OrElseGet(func() int { called = true; return 0 })
	if called {
		t.Errorf("OrElseGet called supplier for a present value")
	}
	if empty.OrElseGet(func() int { return 7 }) != 7 {
		t.Errorf("OrElseGet wrong")
	}
	if empty.Or(func() Optional[int] { return Of(3) }).Get() != 3 || full.Or(Empty[int]).Get() != 5 {
		t.Errorf("Or wrong")
	}
	if full.Filter(func(i int) bool { return i > 10 }).IsPresent() || !full.Filter(func(i int) bool { return i == 5 }).IsPresent() {
		t.Errorf("Filter wrong")
	}
	if p := full.Ptr(); p == nil || *p != 5 || empty.Ptr() != nil {
		t.Errorf("Ptr wrong")
	}
}

func TestIfPresent(t *testing.T) {
	var got []string
	Of("x").IfPresent(func(s string) { got = append(got, s) })
	Empty[string]().IfPresent(func(s string) { got = append(got, s) })
	Empty[string]().IfPresentOrElse(func(string) {}, func() { got = append(got, "else") })
	if len(got) != 2 || got[0] != "x" || got[1] != "else" {
		t.Errorf("got %v", got)
	}
}

func TestMustGetPanicsWhenEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("MustGet on empty did not panic")
		}
	}()
	Empty[int]().MustGet()
}

func TestMapAndFlatMap(t *testing.T) {
	if got := Map(Of(21), func(i int) string { return strconv.Itoa(i * 2) }); got.Get() != "42" {
		t.Errorf("Map = %v", got)
	}
	if Map(Empty[int](), strconv.Itoa).IsPresent() {
		t.Errorf("Map of empty is present")
	}
	parse := func(s string) Optional[int] {
		i, err := strconv.Atoi(s)
		return OfOk(i, err == nil)
	}
	if FlatMap(Of("12"), parse).Get() != 12 || FlatMap(Of("x"), parse).IsPresent() {
		t.Errorf("FlatMap wrong")
	}
}

func TestJSON(t *testing.T) {
	type doc struct {
		A Optional[int]    `json:"a"`
		B Optional[string] `json:"b,omitzero"`
		C Optional[string] `json:"c,omitzero"`
	}
	out, err := json.Marshal(doc{A: Empty[int](), C: Of("hi")})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"a":null,"c":"hi"}` {
		t.Errorf("Marshal = %s", out)
	}

	var d doc
	if err := json.Unmarshal([]byte(`{"a":0,"b":null,"c":"x"}`), &d); err != nil {
		t.Fatal(err)
	}
	if !d.A.IsPresent() || d.A.Get() != 0 || d.B.IsPresent() || d.C.Get() != "x" {
		t.Errorf("Unmarshal = %+v", d)
	}
	if err := json.Unmarshal([]byte(`{"a":"nope"}`), &d); err == nil {
		t.Errorf("Unmarshal accepted a mistyped value")
	}
}
