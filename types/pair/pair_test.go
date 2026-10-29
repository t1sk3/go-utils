package pair

import "testing"

func TestPair(t *testing.T) {
	p := Of("alice", 31)
	if name, age := p.Unpack(); name != "alice" || age != 31 {
		t.Errorf("Unpack() = %v, %v", name, age)
	}
	if s := p.Swap(); s.First != 31 || s.Second != "alice" {
		t.Errorf("Swap() = %v", s)
	}
	if got := p.String(); got != "(alice, 31)" {
		t.Errorf("String() = %q", got)
	}
}

func TestPairIsComparable(t *testing.T) {
	m := map[Pair[string, int]]bool{Of("a", 1): true}
	if !m[Of("a", 1)] || m[Of("a", 2)] {
		t.Errorf("Pair does not work as a map key")
	}
}
