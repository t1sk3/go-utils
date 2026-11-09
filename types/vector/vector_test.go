package vector

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

const eps = 1e-9

func TestVecNConstructors(t *testing.T) {
	src := []int{1, 2, 3}
	v := Of(src...)
	src[0] = 99
	if !v.Equal(VecN[int]{1, 2, 3}) {
		t.Errorf("Of aliases its argument: %v", v)
	}
	if got := Zero[float64](3); !got.Equal(VecN[float64]{0, 0, 0}) || !got.IsZero() {
		t.Errorf("Zero = %v", got)
	}
	if got := Fill(2, 7); !got.Equal(VecN[int]{7, 7}) {
		t.Errorf("Fill = %v", got)
	}
	if got := Unit[int](4, 2); !got.Equal(VecN[int]{0, 0, 1, 0}) {
		t.Errorf("Unit = %v", got)
	}
	if got := Convert[float64](Of(1, 2)); !got.Equal(VecN[float64]{1, 2}) {
		t.Errorf("Convert = %v", got)
	}
	if Of[int]().Dim() != 0 || Convert[float64](VecN[int](nil)) != nil {
		t.Errorf("empty vector")
	}
}

func TestVecNArithmetic(t *testing.T) {
	a, b := Of(1, 2, 3), Of(4, -5, 6)
	cases := []struct {
		name      string
		got, want VecN[int]
	}{
		{"Add", a.Add(b), Of(5, -3, 9)},
		{"Sub", a.Sub(b), Of(-3, 7, -3)},
		{"Mul", a.Mul(b), Of(4, -10, 18)},
		{"Div", b.Div(a), Of(4, -2, 2)},
		{"Scale", a.Scale(3), Of(3, 6, 9)},
		{"DivScalar", b.DivScalar(2), Of(2, -2, 3)},
		{"Neg", a.Neg(), Of(-1, -2, -3)},
		{"Abs", b.Abs(), Of(4, 5, 6)},
		{"Min", a.Min(b), Of(1, -5, 3)},
		{"Max", a.Max(b), Of(4, 2, 6)},
		{"Clamp", Of(-9, 5, 9).Clamp(Of(0, 0, 0), Of(4, 4, 4)), Of(0, 4, 4)},
		{"Reflect", Of(1, -1, 0).Reflect(Of(0, 1, 0)), Of(1, 1, 0)},
	}
	for _, c := range cases {
		if !c.got.Equal(c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if !a.Equal(Of(1, 2, 3)) {
		t.Errorf("operations modified their receiver: %v", a)
	}
	if a.Dot(b) != 12 || a.Sum() != 6 || a.LengthSquared() != 14 || a.DistanceSquared(b) != 9+49+9 {
		t.Errorf("Dot/Sum/LengthSquared/DistanceSquared")
	}
}

func TestVecNMetric(t *testing.T) {
	v := Of(3.0, 4.0)
	if v.Length() != 5 || v.Distance(Of(0.0, 0.0)) != 5 {
		t.Errorf("Length/Distance")
	}
	if got := v.Normalize(); !got.ApproxEqual(Of(0.6, 0.8), eps) {
		t.Errorf("Normalize = %v", got)
	}
	if got := Zero[float64](2).Normalize(); !got.IsZero() {
		t.Errorf("Normalize(0) = %v", got)
	}
	if got := Of(0.0, 0.0).Lerp(Of(10.0, -10.0), 0.25); !got.Equal(Of(2.5, -2.5)) {
		t.Errorf("Lerp = %v", got)
	}
	if got := Of(1.0, 0.0).AngleTo(Of(0.0, 2.0)); math.Abs(got-math.Pi/2) > eps {
		t.Errorf("AngleTo = %v", got)
	}
	if got := Of(1.0, 1.0).AngleTo(Of(2.0, 2.0)); math.IsNaN(got) || got > 1e-7 {
		t.Errorf("AngleTo(parallel) = %v, want ~0", got)
	}
	if got := Of(1.0, 1.0).AngleTo(Of(-3.0, -3.0)); math.IsNaN(got) || math.Abs(got-math.Pi) > 1e-7 {
		t.Errorf("AngleTo(opposite) = %v, want ~π", got)
	}
	if Of(1.0, 0.0).AngleTo(Zero[float64](2)) != 0 {
		t.Errorf("AngleTo(0)")
	}
	if got := Of(2.0, 3.0).Project(Of(5.0, 0.0)); !got.Equal(Of(2.0, 0.0)) {
		t.Errorf("Project = %v", got)
	}
	if got := Of(2.0, 3.0).Project(Zero[float64](2)); !got.IsZero() {
		t.Errorf("Project(0) = %v", got)
	}
	if Of(1.0, 2.0).ApproxEqual(Of(1.0, 2.1), 0.05) || Of(1.0).ApproxEqual(Of(1.0, 0.0), 1) {
		t.Errorf("ApproxEqual")
	}
}

func TestVecNConversions(t *testing.T) {
	if Of(1, 2).ToVec2() != Of2(1, 2) || Of(1, 2, 3).ToVec3() != Of3(1, 2, 3) {
		t.Errorf("ToVec2/ToVec3")
	}
	if got := Of(1, 2).String(); got != "(1, 2)" {
		t.Errorf("String = %q", got)
	}
	if got := Of(1.5).Clone(); !got.Equal(Of(1.5)) {
		t.Errorf("Clone")
	}
	b, _ := json.Marshal(Of(1, 2, 3))
	if string(b) != "[1,2,3]" {
		t.Errorf("JSON = %s", b)
	}
}

func TestVecNDimensionMismatch(t *testing.T) {
	a, b := Of(1, 2), Of(1, 2, 3)
	ops := map[string]func(){
		"Add":      func() { a.Add(b) },
		"Dot":      func() { a.Dot(b) },
		"Distance": func() { a.Distance(b) },
		"AngleTo":  func() { a.AngleTo(b) },
		"Project":  func() { a.Project(b) },
		"ToVec3":   func() { a.ToVec3() },
		"ToVec2":   func() { b.ToVec2() },
	}
	for name, op := range ops {
		func() {
			defer func() {
				r := recover()
				if s, ok := r.(string); !ok || !strings.Contains(s, "dimension mismatch") {
					t.Errorf("%s: recovered %v, want a dimension mismatch panic", name, r)
				}
			}()
			op()
		}()
	}
}

func TestVec2(t *testing.T) {
	a, b := Of2(1, 2), Of2(3, -4)
	cases := []struct {
		name      string
		got, want Vec2[int]
	}{
		{"Add", a.Add(b), Of2(4, -2)},
		{"Sub", a.Sub(b), Of2(-2, 6)},
		{"Mul", a.Mul(b), Of2(3, -8)},
		{"Div", b.Div(a), Of2(3, -2)},
		{"Scale", a.Scale(2), Of2(2, 4)},
		{"DivScalar", b.DivScalar(2), Of2(1, -2)},
		{"Neg", a.Neg(), Of2(-1, -2)},
		{"Abs", b.Abs(), Of2(3, 4)},
		{"Min", a.Min(b), Of2(1, -4)},
		{"Max", a.Max(b), Of2(3, 2)},
		{"Clamp", Of2(-5, 5).Clamp(Of2(0, 0), Of2(3, 3)), Of2(0, 3)},
		{"Perp", a.Perp(), Of2(-2, 1)},
		{"Reflect", Of2(1, -1).Reflect(Of2(0, 1)), Of2(1, 1)},
		{"Lerp", Of2(0, 0).Lerp(Of2(10, 10), 0.5), Of2(5, 5)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if a.Dot(b) != -5 || a.Cross(b) != -10 || a.Sum() != 3 || b.LengthSquared() != 25 {
		t.Errorf("Dot/Cross/Sum/LengthSquared")
	}
	if b.Length() != 5 || a.Distance(a.Add(Of2(3, 4))) != 5 || a.DistanceSquared(Of2(4, 6)) != 25 {
		t.Errorf("Length/Distance")
	}
	if Of2(1, 0).Cross(Of2(0, 1)) <= 0 || Of2(0, 1).Cross(Of2(1, 0)) >= 0 {
		t.Errorf("Cross sign should follow counter-clockwise rotation")
	}

	f := Of2(3.0, 4.0)
	if !f.Normalize().ApproxEqual(Of2(0.6, 0.8), eps) || !(Vec2[float64]{}).Normalize().IsZero() {
		t.Errorf("Normalize")
	}
	if got := Of2(1.0, 0.0).Rotate(math.Pi / 2); !got.ApproxEqual(Of2(0.0, 1.0), eps) {
		t.Errorf("Rotate = %v", got)
	}
	if got := Of2(0.0, 2.0).Angle(); math.Abs(got-math.Pi/2) > eps {
		t.Errorf("Angle = %v", got)
	}
	if got := FromPolar[float64](2, math.Pi); !got.ApproxEqual(Of2(-2.0, 0.0), eps) {
		t.Errorf("FromPolar = %v", got)
	}
	if got := Of2(1.0, 0.0).AngleTo(Of2(-1.0, 0.0)); math.Abs(got-math.Pi) > eps {
		t.Errorf("AngleTo = %v", got)
	}
	if got := Of2(2.0, 3.0).Project(Of2(0.0, 7.0)); got != Of2(0.0, 3.0) {
		t.Errorf("Project = %v", got)
	}
	if !Of2(2, 3).Project(Vec2[int]{}).IsZero() {
		t.Errorf("Project(0)")
	}
	if Convert2[float32](Of2(1, 2)) != Of2[float32](1, 2) {
		t.Errorf("Convert2")
	}
	if a.Extend(3) != Of3(1, 2, 3) || !a.ToN().Equal(Of(1, 2)) || a.Dim() != 2 {
		t.Errorf("Extend/ToN/Dim")
	}
	if a.String() != "(1, 2)" {
		t.Errorf("String = %q", a.String())
	}
}

func TestVec3(t *testing.T) {
	a, b := Of3(1, 2, 3), Of3(4, -5, 6)
	cases := []struct {
		name      string
		got, want Vec3[int]
	}{
		{"Add", a.Add(b), Of3(5, -3, 9)},
		{"Sub", a.Sub(b), Of3(-3, 7, -3)},
		{"Mul", a.Mul(b), Of3(4, -10, 18)},
		{"Div", b.Div(a), Of3(4, -2, 2)},
		{"Scale", a.Scale(2), Of3(2, 4, 6)},
		{"DivScalar", b.DivScalar(2), Of3(2, -2, 3)},
		{"Neg", a.Neg(), Of3(-1, -2, -3)},
		{"Abs", b.Abs(), Of3(4, 5, 6)},
		{"Min", a.Min(b), Of3(1, -5, 3)},
		{"Max", a.Max(b), Of3(4, 2, 6)},
		{"Clamp", Of3(-1, 5, 2).Clamp(Of3(0, 0, 0), Of3(3, 3, 3)), Of3(0, 3, 2)},
		{"Cross", a.Cross(b), Of3(27, 6, -13)},
		{"Cross X×Y", Of3(1, 0, 0).Cross(Of3(0, 1, 0)), Of3(0, 0, 1)},
		{"Reflect", Of3(1, -1, 2).Reflect(Of3(0, 1, 0)), Of3(1, 1, 2)},
		{"Lerp", Of3(0, 0, 0).Lerp(Of3(4, 8, 12), 0.25), Of3(1, 2, 3)},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if c := a.Cross(b); c.Dot(a) != 0 || c.Dot(b) != 0 {
		t.Errorf("Cross is not perpendicular to its operands")
	}
	if a.Dot(b) != 12 || a.Sum() != 6 || a.LengthSquared() != 14 || a.DistanceSquared(b) != 67 {
		t.Errorf("Dot/Sum/LengthSquared/DistanceSquared")
	}
	if Of3(2, 3, 6).Length() != 7 || Of3(1, 1, 1).Distance(Of3(3, 4, 7)) != 7 {
		t.Errorf("Length/Distance")
	}

	if got := Of3(0.0, 3.0, 4.0).Normalize(); !got.ApproxEqual(Of3(0.0, 0.6, 0.8), eps) {
		t.Errorf("Normalize = %v", got)
	}
	if !(Vec3[float64]{}).Normalize().IsZero() {
		t.Errorf("Normalize(0)")
	}
	if got := Of3(1.0, 0.0, 0.0).AngleTo(Of3(0.0, 0.0, 5.0)); math.Abs(got-math.Pi/2) > eps {
		t.Errorf("AngleTo = %v", got)
	}
	if got := Of3(1.0, 2.0, 3.0).Project(Of3(0.0, 0.0, 2.0)); got != Of3(0.0, 0.0, 3.0) {
		t.Errorf("Project = %v", got)
	}
	if !Of3(1, 2, 3).Project(Vec3[int]{}).IsZero() {
		t.Errorf("Project(0)")
	}
	if Convert3[float64](a) != Of3(1.0, 2.0, 3.0) || a.XY() != Of2(1, 2) || !a.ToN().Equal(Of(1, 2, 3)) || a.Dim() != 3 {
		t.Errorf("Convert3/XY/ToN/Dim")
	}
	if a.String() != "(1, 2, 3)" {
		t.Errorf("String = %q", a.String())
	}
}

func TestFixedVectorsAgreeWithVecN(t *testing.T) {
	a2, b2 := Of2(1.5, -2.0), Of2(0.25, 3.0)
	a3, b3 := Of3(1.5, -2.0, 4.0), Of3(0.25, 3.0, -1.0)
	pairs := []struct{ a, b VecN[float64] }{{a2.ToN(), b2.ToN()}, {a3.ToN(), b3.ToN()}}
	fixed := []struct {
		length, dist, angle float64
		norm, proj, lerp    VecN[float64]
	}{
		{a2.Length(), a2.Distance(b2), a2.AngleTo(b2), a2.Normalize().ToN(), a2.Project(b2).ToN(), a2.Lerp(b2, 0.3).ToN()},
		{a3.Length(), a3.Distance(b3), a3.AngleTo(b3), a3.Normalize().ToN(), a3.Project(b3).ToN(), a3.Lerp(b3, 0.3).ToN()},
	}
	for i, p := range pairs {
		f := fixed[i]
		if math.Abs(f.length-p.a.Length()) > eps || math.Abs(f.dist-p.a.Distance(p.b)) > eps ||
			math.Abs(f.angle-p.a.AngleTo(p.b)) > eps || !f.norm.ApproxEqual(p.a.Normalize(), eps) ||
			!f.proj.ApproxEqual(p.a.Project(p.b), eps) || !f.lerp.ApproxEqual(p.a.Lerp(p.b, 0.3), eps) {
			t.Errorf("dimension %d: fixed-size and VecN results differ", p.a.Dim())
		}
	}
}

func TestIntegerResultsTruncate(t *testing.T) {
	if got := Of2(3, 4).Normalize(); got != (Vec2[int]{}) {
		t.Errorf("Normalize of an int vector = %v, want truncation to (0, 0)", got)
	}
	if got := Of2(0, 0).Lerp(Of2(3, 3), 0.5); got != Of2(1, 1) {
		t.Errorf("Lerp = %v, want (1, 1)", got)
	}
}

func TestJSON(t *testing.T) {
	b, _ := json.Marshal(Of3(1, 2, 3))
	if string(b) != `{"x":1,"y":2,"z":3}` {
		t.Errorf("Vec3 JSON = %s", b)
	}
	var v Vec2[float64]
	if err := json.Unmarshal([]byte(`{"x":1.5,"y":-2}`), &v); err != nil || v != Of2(1.5, -2.0) {
		t.Errorf("Vec2 unmarshal = %v, %v", v, err)
	}
}

func TestComparableAsMapKey(t *testing.T) {
	seen := map[Vec2[int]]bool{Of2(1, 2): true}
	if !seen[Of2(0, 0).Add(Of2(1, 2))] {
		t.Errorf("Vec2 not usable as a map key")
	}
}

func BenchmarkVec3AddCrossNormalize(b *testing.B) {
	v, o := Of3(1.0, 2.0, 3.0), Of3(-4.0, 5.0, 0.5)
	b.ReportAllocs()
	for b.Loop() {
		v = v.Add(o).Cross(o).Normalize()
	}
	_ = v
}

func BenchmarkVecNAddDot(b *testing.B) {
	v, o := Fill(16, 1.0), Fill(16, 0.5)
	b.ReportAllocs()
	var s float64
	for b.Loop() {
		s += v.Add(o).Dot(o)
	}
	_ = s
}
