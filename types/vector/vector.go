// Package vector provides Euclidean vectors over any real number type: VecN, a vector
// of any dimension, and Vec2 and Vec3, fixed-size vectors with X, Y and Z fields.
//
// All three are value types. Every operation returns a new vector and leaves its
// receiver unchanged, so vectors can be shared freely. Vec2 and Vec3 are small
// comparable structs that never allocate and can be used as map keys; VecN is a
// slice, so it can be indexed, ranged over and passed to the slices package.
//
// The element type T can be any integer or floating-point type. Arithmetic (Add, Sub,
// Scale, Dot, Cross, ...) stays in T. Operations that are only defined over the reals
// (Length, Distance, AngleTo, ...) return float64, and operations that produce a
// vector from a real-valued computation (Normalize, Lerp, Project, Rotate, ...)
// convert the result back to T, which truncates toward zero for integer types. To
// work with an integer vector in floating point, convert it first:
//
//	grid := vector.Of2(3, 4)                           // Vec2[int]
//	dir := vector.Convert2[float64](grid).Normalize()  // (0.6, 0.8)
//
// Go has no subtyping, so Vec2 and Vec3 are separate types rather than kinds of VecN.
// They have the same methods, and convert to and from VecN with ToN, ToVec2 and ToVec3.
//
// Operations that combine two VecN values panic if their dimensions differ.
// Dividing integer vectors by zero panics, as it does for plain integers.
package vector

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/constraints"
)

// VecN is a vector of any dimension. Its dimension is its length.
//
// Methods that combine two vectors panic if their dimensions differ. A nil VecN is
// the zero-dimensional vector.
type VecN[T constraints.Number] []T

// Of returns a vector with the given components. It copies them, so changing the
// slice passed as xs... does not change the vector.
func Of[T constraints.Number](xs ...T) VecN[T] {
	return VecN[T](slices.Clone(xs))
}

// Zero returns the n-dimensional zero vector.
func Zero[T constraints.Number](n int) VecN[T] {
	return make(VecN[T], n)
}

// Fill returns an n-dimensional vector with every component set to x.
func Fill[T constraints.Number](n int, x T) VecN[T] {
	v := make(VecN[T], n)
	for i := range v {
		v[i] = x
	}
	return v
}

// Unit returns the n-dimensional standard basis vector e_i: 1 at index i and 0
// elsewhere. It panics unless 0 <= i < n.
func Unit[T constraints.Number](n, i int) VecN[T] {
	v := make(VecN[T], n)
	v[i] = 1
	return v
}

// Convert returns v with each component converted to U.
func Convert[U, T constraints.Number](v VecN[T]) VecN[U] {
	if v == nil {
		return nil
	}
	out := make(VecN[U], len(v))
	for i, x := range v {
		out[i] = U(x)
	}
	return out
}

// Dim returns the number of components.
func (v VecN[T]) Dim() int { return len(v) }

// Clone returns a copy of v that does not share storage with it.
func (v VecN[T]) Clone() VecN[T] { return slices.Clone(v) }

// Add returns v + o.
func (v VecN[T]) Add(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return a + b })
}

// Sub returns v - o.
func (v VecN[T]) Sub(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return a - b })
}

// Mul returns the component-wise (Hadamard) product of v and o.
func (v VecN[T]) Mul(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return a * b })
}

// Div returns the component-wise quotient of v and o.
func (v VecN[T]) Div(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return a / b })
}

// Scale returns v multiplied by the scalar s.
func (v VecN[T]) Scale(s T) VecN[T] {
	return v.mapc(func(a T) T { return a * s })
}

// DivScalar returns v divided by the scalar s.
func (v VecN[T]) DivScalar(s T) VecN[T] {
	return v.mapc(func(a T) T { return a / s })
}

// Neg returns -v. For unsigned T the components wrap around, as unary minus does.
func (v VecN[T]) Neg() VecN[T] {
	return v.mapc(func(a T) T { return -a })
}

// Abs returns v with every component replaced by its absolute value.
func (v VecN[T]) Abs() VecN[T] {
	return v.mapc(abs[T])
}

// Min returns the component-wise minimum of v and o.
func (v VecN[T]) Min(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return min(a, b) })
}

// Max returns the component-wise maximum of v and o.
func (v VecN[T]) Max(o VecN[T]) VecN[T] {
	return v.zip(o, func(a, b T) T { return max(a, b) })
}

// Clamp returns v with every component limited to the range given by the matching
// components of lo and hi.
func (v VecN[T]) Clamp(lo, hi VecN[T]) VecN[T] {
	return v.Max(lo).Min(hi)
}

// Sum returns the sum of the components.
func (v VecN[T]) Sum() T {
	var s T
	for _, x := range v {
		s += x
	}
	return s
}

// Dot returns the dot product of v and o.
func (v VecN[T]) Dot(o VecN[T]) T {
	mustMatch(len(v), len(o))
	var s T
	for i := range v {
		s += v[i] * o[i]
	}
	return s
}

// LengthSquared returns the squared Euclidean length of v, v·v. It avoids the square
// root of Length and stays in T, so it suits comparing lengths.
func (v VecN[T]) LengthSquared() T { return v.Dot(v) }

// Length returns the Euclidean length (magnitude) of v.
func (v VecN[T]) Length() float64 {
	var s float64
	for _, x := range v {
		f := float64(x)
		s += f * f
	}
	return math.Sqrt(s)
}

// DistanceSquared returns the squared Euclidean distance between v and o.
func (v VecN[T]) DistanceSquared(o VecN[T]) T {
	mustMatch(len(v), len(o))
	var s T
	for i := range v {
		d := v[i] - o[i]
		s += d * d
	}
	return s
}

// Distance returns the Euclidean distance between v and o.
func (v VecN[T]) Distance(o VecN[T]) float64 {
	mustMatch(len(v), len(o))
	var s float64
	for i := range v {
		d := float64(v[i]) - float64(o[i])
		s += d * d
	}
	return math.Sqrt(s)
}

// Normalize returns the unit vector pointing in the same direction as v. The zero
// vector has no direction, so it is returned unchanged.
func (v VecN[T]) Normalize() VecN[T] {
	l := v.Length()
	if l == 0 {
		return v.Clone()
	}
	return v.mapc(func(a T) T { return T(float64(a) / l) })
}

// Lerp returns the linear interpolation between v (at t = 0) and o (at t = 1).
// Values of t outside [0, 1] extrapolate.
func (v VecN[T]) Lerp(o VecN[T], t float64) VecN[T] {
	return v.zip(o, func(a, b T) T { return lerp(a, b, t) })
}

// AngleTo returns the angle between v and o in radians, in [0, π]. It returns 0 if
// either vector is zero.
func (v VecN[T]) AngleTo(o VecN[T]) float64 {
	mustMatch(len(v), len(o))
	var dot float64
	for i := range v {
		dot += float64(v[i]) * float64(o[i])
	}
	return angle(dot, v.Length()*o.Length())
}

// Project returns the projection of v onto o: the component of v that points along
// o. Projecting onto the zero vector gives the zero vector.
func (v VecN[T]) Project(o VecN[T]) VecN[T] {
	mustMatch(len(v), len(o))
	var num, den float64
	for i := range v {
		num += float64(v[i]) * float64(o[i])
		den += float64(o[i]) * float64(o[i])
	}
	if den == 0 {
		return Zero[T](len(v))
	}
	k := num / den
	return o.mapc(func(a T) T { return T(float64(a) * k) })
}

// Reflect returns v reflected off a surface with normal n, v - 2(v·n)n. The normal
// must have unit length.
func (v VecN[T]) Reflect(n VecN[T]) VecN[T] {
	return v.Sub(n.Scale(2 * v.Dot(n)))
}

// IsZero reports whether every component is zero.
func (v VecN[T]) IsZero() bool {
	for _, x := range v {
		if x != 0 {
			return false
		}
	}
	return true
}

// Equal reports whether v and o have the same dimension and components.
func (v VecN[T]) Equal(o VecN[T]) bool { return slices.Equal(v, o) }

// ApproxEqual reports whether v and o have the same dimension and every pair of
// components differs by at most eps.
func (v VecN[T]) ApproxEqual(o VecN[T], eps float64) bool {
	if len(v) != len(o) {
		return false
	}
	for i := range v {
		if !approx(v[i], o[i], eps) {
			return false
		}
	}
	return true
}

// ToVec2 returns v as a Vec2. It panics unless v has dimension 2.
func (v VecN[T]) ToVec2() Vec2[T] {
	mustMatch(len(v), 2)
	return Vec2[T]{v[0], v[1]}
}

// ToVec3 returns v as a Vec3. It panics unless v has dimension 3.
func (v VecN[T]) ToVec3() Vec3[T] {
	mustMatch(len(v), 3)
	return Vec3[T]{v[0], v[1], v[2]}
}

// String formats v as "(x1, x2, ..., xn)".
func (v VecN[T]) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, x := range v {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprint(&b, x)
	}
	b.WriteByte(')')
	return b.String()
}

func (v VecN[T]) mapc(f func(T) T) VecN[T] {
	if v == nil {
		return nil
	}
	out := make(VecN[T], len(v))
	for i, x := range v {
		out[i] = f(x)
	}
	return out
}

func (v VecN[T]) zip(o VecN[T], f func(a, b T) T) VecN[T] {
	mustMatch(len(v), len(o))
	if v == nil {
		return nil
	}
	out := make(VecN[T], len(v))
	for i := range v {
		out[i] = f(v[i], o[i])
	}
	return out
}

func mustMatch(a, b int) {
	if a != b {
		panic(fmt.Sprintf("vector: dimension mismatch: %d != %d", a, b))
	}
}

func abs[T constraints.Number](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func lerp[T constraints.Number](a, b T, t float64) T {
	fa := float64(a)
	return T(fa + (float64(b)-fa)*t)
}

func approx[T constraints.Number](a, b T, eps float64) bool {
	return math.Abs(float64(a)-float64(b)) <= eps
}

// angle returns acos(dot / lengths), clamped against rounding error, or 0 when
// lengths is 0.
func angle(dot, lengths float64) float64 {
	if lengths == 0 {
		return 0
	}
	return math.Acos(max(-1, min(1, dot/lengths)))
}
