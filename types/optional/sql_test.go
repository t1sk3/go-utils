package optional

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"
)

var (
	_ sql.Scanner   = (*Optional[int])(nil)
	_ sql.Scanner   = (*Null[int])(nil)
	_ driver.Valuer = Null[int]{}
)

func TestScan(t *testing.T) {
	var s Optional[string]
	if err := s.Scan("hello"); err != nil || s != Of("hello") {
		t.Errorf("Scan(string) = %v, %v", s, err)
	}
	if err := s.Scan([]byte("bytes")); err != nil || s != Of("bytes") {
		t.Errorf("Scan([]byte) = %v, %v", s, err)
	}
	if err := s.Scan(nil); err != nil || s.IsPresent() {
		t.Errorf("Scan(nil) = %v, %v", s, err)
	}

	var n Optional[int]
	if err := n.Scan(int64(42)); err != nil || n != Of(42) {
		t.Errorf("Scan(int64) into int = %v, %v", n, err)
	}
	if err := n.Scan("not a number"); err == nil {
		t.Errorf("Scan of an unconvertible value did not fail")
	}

	// T implementing sql.Scanner itself.
	var ns Optional[sql.NullString]
	if err := ns.Scan("x"); err != nil || ns.Get() != (sql.NullString{String: "x", Valid: true}) {
		t.Errorf("Scan into a Scanner type = %v, %v", ns, err)
	}
}

type celsius float64

func TestNullValue(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		got  driver.Valuer
		want driver.Value
	}{
		{"empty", Empty[string]().SQL(), nil},
		{"string", Of("a").SQL(), "a"},
		{"int widened to int64", Of(7).SQL(), int64(7)},
		{"named float", Of(celsius(21.5)).SQL(), 21.5},
		{"time", Of(now).SQL(), now},
		{"inner Valuer", Of(sql.NullInt64{Int64: 3, Valid: true}).SQL(), int64(3)},
	}
	for _, tt := range tests {
		got, err := tt.got.Value()
		if err != nil || got != tt.want {
			t.Errorf("%s: Value() = %#v, %v; want %#v", tt.name, got, err, tt.want)
		}
	}
	if _, err := Of(struct{}{}).SQL().Value(); err == nil {
		t.Errorf("Value() of an unsupported type did not fail")
	}
}

func TestNullKeepsOptionalBehaviour(t *testing.T) {
	var n Null[int]
	if err := n.Scan(int64(5)); err != nil || n.OrElse(0) != 5 {
		t.Errorf("Scan into Null = %v, %v", n, err)
	}
	if v, ok := n.Optional.Value(); v != 5 || !ok {
		t.Errorf("n.Optional.Value() = %v, %v", v, ok)
	}

	type row struct {
		Email Null[string] `json:",omitzero"`
	}
	if data, _ := json.Marshal(row{}); string(data) != "{}" {
		t.Errorf("omitzero on empty Null = %s", data)
	}
	if data, _ := json.Marshal(row{Of("a@b").SQL()}); string(data) != `{"Email":"a@b"}` {
		t.Errorf("Marshal Null = %s", data)
	}
	var r row
	if err := json.Unmarshal([]byte(`{"Email":"x"}`), &r); err != nil || r.Email.Get() != "x" {
		t.Errorf("Unmarshal Null = %v, %v", r, err)
	}
}
