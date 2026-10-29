package constraints

import (
	"testing"
	"time"
)

// These helpers only compile when each type argument is in the constraint's type
// set, so the test is mostly checked by the compiler.

func neg[T SignedNumber](v T) T         { return -v }
func add[T Numeric](a, b T) T           { return a + b }
func less[T Number](a, b T) bool        { return a < b }
func eq[T Basic](a, b T) bool           { return a == b }
func mod[T Integer](a, b T) T           { return a % b }
func square[T Complex](v T) T           { return v * v }
func isZero[T Float](v T) bool          { return v == 0 }
func double[T Unsigned | Signed](v T) T { return v * 2 }

type celsius float64

func TestConstraints(t *testing.T) {
	if neg(int8(3)) != -3 || neg(celsius(1.5)) != -1.5 || neg(complex(1, 1)) != complex(-1, -1) {
		t.Errorf("SignedNumber")
	}
	if add(uint(2), 3) != 5 || add(complex64(1), 2i) != 1+2i || add(time.Second, time.Second) != 2*time.Second {
		t.Errorf("Numeric")
	}
	if !less(1.5, 2) || !less(uintptr(1), 2) {
		t.Errorf("Number")
	}
	if !eq("a", "a") || !eq(true, true) || !eq(1i, 1i) || eq(celsius(1), 2) {
		t.Errorf("Basic")
	}
	if mod(uint16(7), 4) != 3 || double(int64(4)) != 8 || square(1i) != -1 || !isZero(float32(0)) {
		t.Errorf("Integer, Complex or Float")
	}
}
