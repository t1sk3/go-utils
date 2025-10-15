package utils

import (
	"fmt"
	"testing"
)

func TestOf_Int(t *testing.T) {
	opt := Of(123)
	if !opt.IsPresent() {
		t.Fatalf("expected present")
	}
	if got := opt.Get(); got != 123 {
		t.Errorf("Get() = %v, want 123", got)
	}
	if s := fmt.Sprint(opt); s != "Optional[123]" {
		t.Errorf("String() = %q, want %q", s, "Optional[123]")
	}
}

func TestOf_StringZero(t *testing.T) {
	opt := Of("")
	if !opt.IsPresent() {
		t.Fatalf("expected present")
	}
	if got := opt.Get(); got != "" {
		t.Errorf("Get() = %q, want empty string", got)
	}
	if s := fmt.Sprint(opt); s != "Optional[]" {
		t.Errorf("String() = %q, want %q", s, "Optional[]")
	}
}

func TestOfNullable_Nil(t *testing.T) {
	var p *int
	opt := OfNullable[int](p)
	if opt.IsPresent() {
		t.Fatalf("expected empty")
	}
	if got := opt.Get(); got != 0 {
		t.Errorf("Get() = %v, want 0", got)
	}
	if s := fmt.Sprint(opt); s != "Optional.empty" {
		t.Errorf("String() = %q, want %q", s, "Optional.empty")
	}
}

func TestOfNullable_NonNil(t *testing.T) {
	v := "abc"
	opt := OfNullable(&v)
	if !opt.IsPresent() {
		t.Fatalf("expected present")
	}
	if got := opt.Get(); got != "abc" {
		t.Errorf("Get() = %q, want %q", got, "abc")
	}
	if s := fmt.Sprint(opt); s != "Optional[abc]" {
		t.Errorf("String() = %q, want %q", s, "Optional[abc]")
	}
	// Verify pointer semantics (value reflects changes to underlying variable)
	v = "def"
	if got := opt.Get(); got != "def" {
		t.Errorf("Get() after change = %q, want %q", got, "def")
	}
}

func TestEmpty_Int(t *testing.T) {
	opt := Empty[int]()
	if opt.IsPresent() {
		t.Fatalf("expected empty")
	}
	if got := opt.Get(); got != 0 {
		t.Errorf("Get() = %v, want 0", got)
	}
	if s := fmt.Sprint(opt); s != "Optional.empty" {
		t.Errorf("String() = %q, want %q", s, "Optional.empty")
	}
}

func TestOfNullable_ZeroValuePresent(t *testing.T) {
	v := 0
	opt := OfNullable(&v)
	if !opt.IsPresent() {
		t.Fatalf("expected present")
	}
	if got := opt.Get(); got != 0 {
		t.Errorf("Get() = %v, want 0", got)
	}
	if s := fmt.Sprint(opt); s != "Optional[0]" {
		t.Errorf("String() = %q, want %q", s, "Optional[0]")
	}
}
