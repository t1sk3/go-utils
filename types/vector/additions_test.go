package vector

import (
	"encoding/json"
	"math"
	"testing"
)

// Every Vec4 method must give what the same method on VecN gives.
func TestVec4AgreesWithVecN(t *testing.T) {
	a, b := Of4(1.5, -2, 3, 0.5), Of4(-4, 0.25, 2, 8)
	lo, hi := Of4(-1.0, -1, -1, -1), Of4(2.0, 2, 2, 2)
	an, bn := a.ToN(), b.ToN()
	vectors := map[string][2]VecN[float64]{
		"Add":         {a.Add(b).ToN(), an.Add(bn)},
		"Sub":         {a.Sub(b).ToN(), an.Sub(bn)},
		"Mul":         {a.Mul(b).ToN(), an.Mul(bn)},
		"Div":         {b.Div(a).ToN(), bn.Div(an)},
		"Scale":       {a.Scale(3).ToN(), an.Scale(3)},
		"DivScalar":   {a.DivScalar(4).ToN(), an.DivScalar(4)},
		"Neg":         {a.Neg().ToN(), an.Neg()},
		"Abs":         {a.Abs().ToN(), an.Abs()},
		"Min":         {a.Min(b).ToN(), an.Min(bn)},
		"Max":         {a.Max(b).ToN(), an.Max(bn)},
		"Clamp":       {a.Clamp(lo, hi).ToN(), an.Clamp(lo.ToN(), hi.ToN())},
		"Normalize":   {a.Normalize().ToN(), an.Normalize()},
		"Lerp":        {a.Lerp(b, 0.3).ToN(), an.Lerp(bn, 0.3)},
		"Slerp":       {a.Slerp(b, 0.3).ToN(), an.Slerp(bn, 0.3)},
		"Project":     {a.Project(b).ToN(), an.Project(bn)},
		"Reflect":     {a.Reflect(b.Normalize()).ToN(), an.Reflect(bn.Normalize())},
		"ClampLength": {a.ClampLength(2).ToN(), an.ClampLength(2)},
		"Floor":       {a.Floor().ToN(), an.Floor()},
		"Ceil":        {a.Ceil().ToN(), an.Ceil()},
		"Round":       {a.Round().ToN(), an.Round()},
	}
	for name, v := range vectors {
		if !v[0].ApproxEqual(v[1], 1e-12) {
			t.Errorf("%s: Vec4 gives %v, VecN gives %v", name, v[0], v[1])
		}
	}
	scalars := map[string][2]float64{
		"Sum":             {a.Sum(), an.Sum()},
		"Dot":             {a.Dot(b), an.Dot(bn)},
		"LengthSquared":   {a.LengthSquared(), an.LengthSquared()},
		"Length":          {a.Length(), an.Length()},
		"DistanceSquared": {a.DistanceSquared(b), an.DistanceSquared(bn)},
		"Distance":        {a.Distance(b), an.Distance(bn)},
		"AngleTo":         {a.AngleTo(b), an.AngleTo(bn)},
	}
	for name, s := range scalars {
		if math.Abs(s[0]-s[1]) > 1e-12 {
			t.Errorf("%s: Vec4 gives %v, VecN gives %v", name, s[0], s[1])
		}
	}
}

func TestVec4(t *testing.T) {
	v := Of4(1, 2, 3, 4)
	if v.Dim() != 4 || v.String() != "(1, 2, 3, 4)" || v.IsZero() || !(Vec4[int]{}).IsZero() {
		t.Errorf("basic properties of %v are wrong", v)
	}
	if v.XYZ() != Of3(1, 2, 3) || Of3(1, 2, 3).Extend(4) != v || v.ToN().ToVec4() != v {
		t.Errorf("conversions do not round-trip")
	}
	if got := Convert4[float64](v); got != Of4(1.0, 2, 3, 4) {
		t.Errorf("Convert4 = %v", got)
	}
	if !Of4(1.0, 2, 3, 4).ApproxEqual(Of4(1.0, 2, 3, 4.05), 0.1) || Of4(1.0, 2, 3, 4).ApproxEqual(Of4(1.0, 2, 3, 5), 0.1) {
		t.Errorf("ApproxEqual is wrong")
	}
	zero := Vec4[float64]{}
	if zero.Normalize() != zero || Of4(1.0, 2, 3, 4).Project(zero) != zero || Of4(1.0, 0, 0, 0).ClampLength(5) != Of4(1.0, 0, 0, 0) {
		t.Errorf("the zero vector or a short vector is not handled")
	}
	if got := Of4(1e200, 0, 0, 1e200).Length(); math.IsInf(got, 0) {
		t.Errorf("Length overflowed")
	}
	data, err := json.Marshal(v)
	var back Vec4[int]
	if err != nil || string(data) != `{"x":1,"y":2,"z":3,"w":4}` || json.Unmarshal(data, &back) != nil || back != v {
		t.Errorf("JSON = %s, %v, back %v", data, err, back)
	}
	seen := map[Vec4[int]]bool{v: true}
	if !seen[Of4(1, 2, 3, 4)] {
		t.Errorf("a Vec4 does not work as a map key")
	}
	defer func() {
		if recover() == nil {
			t.Errorf("ToVec4 of a 3D vector did not panic")
		}
	}()
	Of(1, 2, 3).ToVec4()
}

func TestSlerp(t *testing.T) {
	x, y := Of3(1.0, 0, 0), Of3(0.0, 1, 0)
	// Halfway along the arc the vector still has unit length, where Lerp cuts the corner.
	mid := x.Slerp(y, 0.5)
	if h := math.Sqrt2 / 2; !mid.ApproxEqual(Of3(h, h, 0), 1e-12) {
		t.Errorf("Slerp halfway between X and Y = %v", mid)
	}
	if l := x.Lerp(y, 0.5).Length(); l > 0.75 {
		t.Errorf("Lerp halfway has length %v, expected the shortcut", l)
	}
	if !x.Slerp(y, 0).ApproxEqual(x, 1e-12) || !x.Slerp(y, 1).ApproxEqual(y, 1e-12) {
		t.Errorf("Slerp does not start and end at its arguments")
	}
	// The length changes linearly while the direction turns evenly.
	long := y.Scale(3)
	third := x.Slerp(long, 1.0/3)
	if math.Abs(third.Length()-(1+2.0/3)) > 1e-12 || math.Abs(x.AngleTo(third)-math.Pi/6) > 1e-12 {
		t.Errorf("Slerp a third of the way = %v, length %v, angle %v", third, third.Length(), x.AngleTo(third))
	}
	// No arc is defined for these, so Slerp falls back to Lerp.
	for name, o := range map[string]Vec3[float64]{"zero": {}, "parallel": x.Scale(4), "opposite": x.Neg()} {
		if got, want := x.Slerp(o, 0.25), x.Lerp(o, 0.25); !got.ApproxEqual(want, 1e-12) {
			t.Errorf("Slerp to a %s vector = %v, want Lerp's %v", name, got, want)
		}
	}
	if got := (Vec3[float64]{}).Slerp(x, 0.5); !got.ApproxEqual(Of3(0.5, 0, 0), 1e-12) {
		t.Errorf("Slerp from the zero vector = %v", got)
	}
	// Vec2 and VecN follow the same arc.
	if got := x.XY().Slerp(y.XY(), 0.5); !got.ApproxEqual(mid.XY(), 1e-12) {
		t.Errorf("Vec2.Slerp = %v", got)
	}
	if got := x.ToN().Slerp(y.ToN(), 0.5); !got.ApproxEqual(mid.ToN(), 1e-12) {
		t.Errorf("VecN.Slerp = %v", got)
	}
	if got := Of2(1.0, 0).Slerp(Of2(-1.0, 0), 0.5); got != (Vec2[float64]{}) {
		t.Errorf("Vec2.Slerp between opposite vectors = %v, want Lerp's zero", got)
	}
	if got := Of(1.0, 0).Slerp(Of(2.0, 0), 0.5); !got.Equal(Of(1.5, 0)) {
		t.Errorf("VecN.Slerp between parallel vectors = %v", got)
	}
}

func TestVec3Rotate(t *testing.T) {
	x, y, z := Of3(1.0, 0, 0), Of3(0.0, 1, 0), Of3(0.0, 0, 1)
	quarter := math.Pi / 2
	tests := []struct {
		name          string
		v, axis, want Vec3[float64]
	}{
		{"X around Z", x, z, y},
		{"Y around Z", y, z, x.Neg()},
		{"Y around X", y, x, z},
		{"Z around Y", z, y, x},
		{"along the axis", z.Scale(3), z, z.Scale(3)},
		{"a long axis", x, z.Scale(10), y},
		{"a negative axis", x, z.Neg(), y.Neg()},
	}
	for _, tt := range tests {
		if got := tt.v.Rotate(tt.axis, quarter); !got.ApproxEqual(tt.want, 1e-12) {
			t.Errorf("%s: Rotate = %v, want %v", tt.name, got, tt.want)
		}
	}
	v, axis := Of3(1.0, 2, 3), Of3(1.0, 1, 1)
	if got := v.Rotate(axis, 2*math.Pi/3); !got.ApproxEqual(Of3(3.0, 1, 2), 1e-12) {
		t.Errorf("a third of a turn around the diagonal = %v, want the components shifted", got)
	}
	r := v.Rotate(axis, 0.7)
	if math.Abs(r.Length()-v.Length()) > 1e-12 || math.Abs(r.Dot(axis)-v.Dot(axis)) > 1e-12 {
		t.Errorf("Rotate changed the length or the component along the axis: %v", r)
	}
	if !r.Rotate(axis, -0.7).ApproxEqual(v, 1e-12) {
		t.Errorf("rotating back does not return the original")
	}
	if v.Rotate(Vec3[float64]{}, 1) != v {
		t.Errorf("rotating around the zero vector changed the vector")
	}
}

func TestClampLengthAndRounding(t *testing.T) {
	if got := Of2(3.0, 4).ClampLength(2.5); !got.ApproxEqual(Of2(1.5, 2), 1e-12) {
		t.Errorf("Vec2.ClampLength = %v", got)
	}
	if got := Of3(0.0, 3, 4).ClampLength(10); got != Of3(0.0, 3, 4) {
		t.Errorf("ClampLength lengthened a short vector: %v", got)
	}
	if got := Of3(0.0, 3, 4).ClampLength(1); !got.ApproxEqual(Of3(0, 0.6, 0.8), 1e-12) {
		t.Errorf("Vec3.ClampLength = %v", got)
	}
	if got := Of(3.0, 4).ClampLength(-1); !got.Equal(Of(0.0, 0)) {
		t.Errorf("ClampLength with a negative limit = %v, want the zero vector", got)
	}
	short := Of(0.3, 0.4)
	clamped := short.ClampLength(1)
	clamped[0] = 9
	if short[0] != 0.3 {
		t.Errorf("VecN.ClampLength returned its receiver's storage")
	}
	if got := Of(math.NaN(), 1).ClampLength(1); !math.IsNaN(got[0]) {
		t.Errorf("ClampLength of a NaN vector = %v", got)
	}

	v := Of3(1.5, -1.5, 2.4)
	if v.Floor() != Of3(1.0, -2, 2) || v.Ceil() != Of3(2.0, -1, 3) || v.Round() != Of3(2.0, -2, 2) {
		t.Errorf("rounding %v = %v, %v, %v", v, v.Floor(), v.Ceil(), v.Round())
	}
	if got := Of2(1.5, -0.5).Floor(); got != Of2(1.0, -1) {
		t.Errorf("Vec2.Floor = %v", got)
	}
	if Of2(1.5, -0.5).Ceil() != Of2(2.0, 0) || Of2(2.5, -0.5).Round() != Of2(3.0, -1) {
		t.Errorf("Vec2 rounding is wrong")
	}
	// An integer that float64 cannot hold exactly must come through unchanged.
	big := int64(1<<62 + 1)
	if got := Of2(big, -big); got.Floor() != got || got.Ceil() != got || got.Round() != got {
		t.Errorf("rounding changed an integer vector")
	}
	if got := Of(big).Round(); got[0] != big {
		t.Errorf("VecN.Round changed an integer: %d", got[0])
	}
	if got := Of3[float32](1.5, 2.5, -0.4).Round(); got != Of3[float32](2, 3, -0) {
		t.Errorf("float32 rounding = %v", got)
	}
}

func TestInPlace(t *testing.T) {
	v := Of(1, 2, 3)
	if got := v.AddInPlace(Of(10, 20, 30)); !v.Equal(Of(11, 22, 33)) || &got[0] != &v[0] {
		t.Errorf("AddInPlace = %v, or it did not return its receiver", v)
	}
	v.SubInPlace(Of(1, 2, 3)).ScaleInPlace(2)
	if !v.Equal(Of(20, 40, 60)) {
		t.Errorf("after SubInPlace and ScaleInPlace: %v", v)
	}
	if allocs := testing.AllocsPerRun(10, func() { v.AddInPlace(v).ScaleInPlace(1) }); allocs != 0 {
		t.Errorf("the in-place methods allocate %v times", allocs)
	}
	for name, f := range map[string]func(){
		"AddInPlace": func() { v.AddInPlace(Of(1)) },
		"SubInPlace": func() { v.SubInPlace(Of(1)) },
		"Slerp":      func() { Of(1.0).Slerp(Of(1.0, 2), 0.5) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s with mismatched dimensions did not panic", name)
				}
			}()
			f()
		}()
	}
}
