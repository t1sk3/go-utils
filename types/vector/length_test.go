package vector

import (
	"math"
	"testing"
)

// Length and Distance must not overflow or underflow while squaring components whose
// length is representable, and Vec2, Vec3 and VecN must agree.
func TestLengthAtExtremes(t *testing.T) {
	for _, x := range []float64{1e200, 1e-200, 3} {
		want := math.Hypot(3*x/3, 4*x/3)
		got := map[string]float64{
			"Vec2":          Of2(3*x/3, 4*x/3).Length(),
			"Vec3":          Of3(3*x/3, 0, 4*x/3).Length(),
			"VecN":          Of(3*x/3, 0, 4*x/3, 0).Length(),
			"Vec3.Distance": Of3(3*x/3, 0, 4*x/3).Distance(Vec3[float64]{}),
			"VecN.Distance": Of(3*x/3, 4*x/3).Distance(Zero[float64](2)),
		}
		for name, l := range got {
			if math.Abs(l-want) > want*1e-14 {
				t.Errorf("%s of scale %g = %g, want %g", name, x, l, want)
			}
		}
	}
	if got := Of3(1e-200, 0, 0).Normalize(); got != Of3(1.0, 0, 0) {
		t.Errorf("Normalize of a tiny Vec3 = %v", got)
	}
	if got := Of(0, 1e-200, 0).Normalize(); !got.Equal(Of(0.0, 1, 0)) {
		t.Errorf("Normalize of a tiny VecN = %v", got)
	}
	if got := Of(0.0, 0, 0).Length(); got != 0 {
		t.Errorf("Length of the zero vector = %v", got)
	}
	if got := Of(1, math.Inf(-1), 2).Length(); !math.IsInf(got, 1) {
		t.Errorf("Length with an infinite component = %v", got)
	}
	// Infinity wins wherever it stands and however often, as it does in math.Hypot.
	inf := math.Inf(1)
	for _, v := range []VecN[float64]{{inf, inf}, {inf, -inf, 1}, {1, inf, inf, 2}, {inf, math.NaN()}, {math.NaN(), inf}, {0, inf}} {
		if got := v.Length(); !math.IsInf(got, 1) {
			t.Errorf("Length of %v = %v, want +Inf", v, got)
		}
		if got := v.Distance(Zero[float64](len(v))); !math.IsInf(got, 1) {
			t.Errorf("Distance of %v from the origin = %v, want +Inf", v, got)
		}
	}
	if got, want := Of3(inf, inf, 0).Length(), Of2(inf, inf).Length(); got != want {
		t.Errorf("Vec3 and Vec2 disagree on an infinite length: %v, %v", got, want)
	}
	if got := Of(1, math.NaN(), 2).Length(); !math.IsNaN(got) {
		t.Errorf("Length with a NaN component = %v", got)
	}
	if got := Of(math.NaN(), 1).Length(); !math.IsNaN(got) {
		t.Errorf("Length with a leading NaN component = %v", got)
	}
	if got := Of(3, 4).Length(); got != 5 {
		t.Errorf("Length of an integer vector = %v", got)
	}
}
