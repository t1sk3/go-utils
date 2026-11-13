package triple

import "testing"

func TestTriple(t *testing.T) {
	tr := Of("a", 2, 3.5)
	if a, b, c := tr.Unpack(); a != "a" || b != 2 || c != 3.5 {
		t.Errorf("Unpack() = %v, %v, %v", a, b, c)
	}
	if tr.String() != "(a, 2, 3.5)" {
		t.Errorf("String() = %q", tr)
	}
	if tr != Of("a", 2, 3.5) || tr == Of("a", 2, 4.5) {
		t.Errorf("== does not compare all three values")
	}
	seen := map[Triple[int, int, int]]bool{Of(1, 2, 3): true}
	if !seen[Of(1, 2, 3)] {
		t.Errorf("a Triple does not work as a map key")
	}
}
