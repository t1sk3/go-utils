package stream

import "testing"

func TestAdd(t *testing.T) {
	if got := Of(1, 2, 3, 4).Fold(0, Add); got != 10 {
		t.Errorf("Fold(0, Add) = %d, want 10", got)
	}
	if got := Of(1.5, 2.25).Fold(0, Add); got != 3.75 {
		t.Errorf("Fold(0, Add) over floats = %v, want 3.75", got)
	}
	if got := Empty[int]().Fold(0, Add); got != 0 {
		t.Errorf("Fold(0, Add) over empty = %d, want 0", got)
	}
	if got := Of(1, 2, 3).Reduce(Add).Get(); got != 6 {
		t.Errorf("Reduce(Add) = %d, want 6", got)
	}
	if got := Of(1, 2, 3).Scan(0, Add).ToSlice(); len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 6 {
		t.Errorf("Scan(0, Add) = %v, want [1 3 6]", got)
	}
	if got := Of(complex(1, 2), complex(3, -1)).Fold(0, Add); got != complex(4, 1) {
		t.Errorf("Fold(0, Add) over complex = %v, want (4+1i)", got)
	}
}

func TestMultiply(t *testing.T) {
	if got := RangeClosed(1, 5).Fold(1, Multiply); got != 120 {
		t.Errorf("Fold(1, Multiply) = %d, want 120", got)
	}
	if got := Empty[int]().Fold(1, Multiply); got != 1 {
		t.Errorf("Fold(1, Multiply) over empty = %d, want 1", got)
	}
	if got := Of(1, 2, 3, 4).Reduce(Multiply).Get(); got != 24 {
		t.Errorf("Reduce(Multiply) = %d, want 24", got)
	}
	if got := Of(0.5, 4.0).Fold(1, Multiply); got != 2 {
		t.Errorf("Fold(1, Multiply) over floats = %v, want 2", got)
	}
	// Integer results wrap around like Go's *.
	if got := Of[int8](16, 16).Fold(1, Multiply); got != 0 {
		t.Errorf("Fold(1, Multiply) over int8 = %d, want 0 (wrapped)", got)
	}
}

func TestOperatorsMatchSum(t *testing.T) {
	if got, want := Range(0, 100).Fold(0, Add), Sum(Range(0, 100)); got != want {
		t.Errorf("Fold(0, Add) = %d, Sum = %d", got, want)
	}
}
