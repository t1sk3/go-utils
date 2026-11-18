package triple

import "testing"

func TestTriple(t *testing.T) {
	tr := Of("alice", 31, true)
	if name, age, ok := tr.Unpack(); name != "alice" || age != 31 || !ok {
		t.Errorf("Unpack() = %v, %v, %v", name, age, ok)
	}
	if r := tr.Reverse(); r.First != true || r.Second != 31 || r.Third != "alice" {
		t.Errorf("Reverse() = %v", r)
	}
	if got := tr.String(); got != "(alice, 31, true)" {
		t.Errorf("String() = %q", got)
	}
}

func TestTripleIsComparable(t *testing.T) {
	m := map[Triple[string, int, bool]]bool{Of("a", 1, true): true}
	if !m[Of("a", 1, true)] || m[Of("a", 1, false)] {
		t.Errorf("Triple does not work as a map key")
	}
}
