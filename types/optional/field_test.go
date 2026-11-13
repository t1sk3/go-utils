package optional

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestFieldStates(t *testing.T) {
	var absent Field[int]
	null, value := NullField[int](), FieldOf(0)
	if !absent.IsAbsent() || absent.IsNull() || absent.HasValue() || !absent.IsZero() {
		t.Errorf("the zero value is not absent: %v", absent)
	}
	if null.IsAbsent() || !null.IsNull() || null.HasValue() || null.IsZero() {
		t.Errorf("NullField is not null: %v", null)
	}
	if value.IsAbsent() || value.IsNull() || !value.HasValue() || value.IsZero() {
		t.Errorf("FieldOf(0) does not hold a value: %v", value)
	}
	if _, ok := absent.Value(); ok {
		t.Errorf("an absent Field has a value")
	}
	if _, ok := null.Value(); ok || null.Optional().IsPresent() {
		t.Errorf("a null Field has a value")
	}
	if v, ok := FieldOf(7).Value(); !ok || v != 7 || FieldOf(7).Optional().OrElse(0) != 7 {
		t.Errorf("FieldOf(7) = %v, %v", v, ok)
	}
	if absent.String() != "Field.absent" || null.String() != "Field.null" || FieldOf(7).String() != "Field[7]" {
		t.Errorf("String() = %q, %q, %q", absent, null, FieldOf(7))
	}
}

func TestFieldJSON(t *testing.T) {
	type patch struct {
		Name  Field[string] `json:"name,omitzero"`
		Email Field[string] `json:"email,omitzero"`
		Age   Field[int]    `json:"age,omitzero"`
	}
	var p patch
	if err := json.Unmarshal([]byte(`{"name":"Ada","email": null }`), &p); err != nil {
		t.Fatal(err)
	}
	if v, ok := p.Name.Value(); !ok || v != "Ada" {
		t.Errorf("a field with a value decoded as %v", p.Name)
	}
	if !p.Email.IsNull() {
		t.Errorf("a null field decoded as %v", p.Email)
	}
	if !p.Age.IsAbsent() {
		t.Errorf("a missing field decoded as %v", p.Age)
	}
	data, err := json.Marshal(p)
	if err != nil || string(data) != `{"name":"Ada","email":null}` {
		t.Errorf("Marshal = %s, %v", data, err)
	}
	// Without omitzero an absent Field has to be written as something.
	if data, _ := json.Marshal(struct{ A Field[int] }{}); string(data) != `{"A":null}` {
		t.Errorf("Marshal of an absent Field without omitzero = %s", data)
	}
	if err := json.Unmarshal([]byte(`{"age":"old"}`), &p); err == nil {
		t.Errorf("Unmarshal of the wrong type did not fail")
	}
	// Decoding over an existing value replaces it.
	if err := json.Unmarshal([]byte(`{"name":null}`), &p); err != nil || !p.Name.IsNull() {
		t.Errorf("decoding null over a value left %v, %v", p.Name, err)
	}
}

func TestEqual(t *testing.T) {
	if !Equal(Of(1), Of(1)) || Equal(Of(1), Of(2)) || Equal(Of(0), Empty[int]()) || !Equal(Empty[int](), Empty[int]()) {
		t.Errorf("Equal is wrong")
	}
	a, b := Of([]int{1, 2}), Of([]int{1, 2})
	if !EqualFunc(a, b, slices.Equal) || EqualFunc(a, Of([]int{9}), slices.Equal) {
		t.Errorf("EqualFunc is wrong for present values")
	}
	if EqualFunc(a, Empty[[]int](), slices.Equal) || !EqualFunc(Empty[[]int](), Empty[[]int](), slices.Equal) {
		t.Errorf("EqualFunc is wrong for empty values")
	}
}
