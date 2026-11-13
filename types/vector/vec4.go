package vector

import (
	"fmt"
	"math"

	"github.com/t1sk3/go-utils/types/constraints"
)

// Vec4 is a four-dimensional vector, as used for homogeneous coordinates and RGBA
// colours. It is a plain comparable struct: the zero value is the zero vector, ==
// compares components, and it never allocates.
type Vec4[T constraints.Number] struct {
	X T `json:"x"`
	Y T `json:"y"`
	Z T `json:"z"`
	W T `json:"w"`
}

// Of4 returns the vector (x, y, z, w).
func Of4[T constraints.Number](x, y, z, w T) Vec4[T] {
	return Vec4[T]{x, y, z, w}
}

// Convert4 returns v with each component converted to U.
func Convert4[U, T constraints.Number](v Vec4[T]) Vec4[U] {
	return Vec4[U]{U(v.X), U(v.Y), U(v.Z), U(v.W)}
}

// Dim returns 4.
func (v Vec4[T]) Dim() int { return 4 }

// each returns the vector of f applied to every component.
func (v Vec4[T]) each(f func(T) T) Vec4[T] {
	return Vec4[T]{f(v.X), f(v.Y), f(v.Z), f(v.W)}
}

// with returns the vector of f applied to every pair of matching components.
func (v Vec4[T]) with(o Vec4[T], f func(a, b T) T) Vec4[T] {
	return Vec4[T]{f(v.X, o.X), f(v.Y, o.Y), f(v.Z, o.Z), f(v.W, o.W)}
}

// floats returns the components as float64.
func (v Vec4[T]) floats() (x, y, z, w float64) {
	return float64(v.X), float64(v.Y), float64(v.Z), float64(v.W)
}

// Add returns v + o.
func (v Vec4[T]) Add(o Vec4[T]) Vec4[T] { return Vec4[T]{v.X + o.X, v.Y + o.Y, v.Z + o.Z, v.W + o.W} }

// Sub returns v - o.
func (v Vec4[T]) Sub(o Vec4[T]) Vec4[T] { return Vec4[T]{v.X - o.X, v.Y - o.Y, v.Z - o.Z, v.W - o.W} }

// Mul returns the component-wise (Hadamard) product of v and o.
func (v Vec4[T]) Mul(o Vec4[T]) Vec4[T] { return Vec4[T]{v.X * o.X, v.Y * o.Y, v.Z * o.Z, v.W * o.W} }

// Div returns the component-wise quotient of v and o.
func (v Vec4[T]) Div(o Vec4[T]) Vec4[T] { return Vec4[T]{v.X / o.X, v.Y / o.Y, v.Z / o.Z, v.W / o.W} }

// Scale returns v multiplied by the scalar s.
func (v Vec4[T]) Scale(s T) Vec4[T] { return Vec4[T]{v.X * s, v.Y * s, v.Z * s, v.W * s} }

// DivScalar returns v divided by the scalar s.
func (v Vec4[T]) DivScalar(s T) Vec4[T] { return Vec4[T]{v.X / s, v.Y / s, v.Z / s, v.W / s} }

// Neg returns -v. For unsigned T the components wrap around, as unary minus does.
func (v Vec4[T]) Neg() Vec4[T] { return Vec4[T]{-v.X, -v.Y, -v.Z, -v.W} }

// Abs returns v with every component replaced by its absolute value. The smallest
// value of a signed integer type has no positive counterpart and stays as it is, as
// with unary minus.
func (v Vec4[T]) Abs() Vec4[T] { return v.each(abs[T]) }

// Min returns the component-wise minimum of v and o.
func (v Vec4[T]) Min(o Vec4[T]) Vec4[T] { return v.with(o, func(a, b T) T { return min(a, b) }) }

// Max returns the component-wise maximum of v and o.
func (v Vec4[T]) Max(o Vec4[T]) Vec4[T] { return v.with(o, func(a, b T) T { return max(a, b) }) }

// Clamp returns v with every component limited to the range given by the matching
// components of lo and hi.
func (v Vec4[T]) Clamp(lo, hi Vec4[T]) Vec4[T] { return v.Max(lo).Min(hi) }

// Sum returns X + Y + Z + W.
func (v Vec4[T]) Sum() T { return v.X + v.Y + v.Z + v.W }

// Dot returns the dot product of v and o.
func (v Vec4[T]) Dot(o Vec4[T]) T { return v.X*o.X + v.Y*o.Y + v.Z*o.Z + v.W*o.W }

// LengthSquared returns the squared Euclidean length of v, v·v. It avoids the square
// root of Length and stays in T, so it suits comparing lengths.
func (v Vec4[T]) LengthSquared() T { return v.Dot(v) }

// Length returns the Euclidean length (magnitude) of v.
func (v Vec4[T]) Length() float64 {
	x, y, z, w := v.floats()
	return math.Hypot(math.Hypot(x, y), math.Hypot(z, w))
}

// DistanceSquared returns the squared Euclidean distance between v and o.
func (v Vec4[T]) DistanceSquared(o Vec4[T]) T { return v.Sub(o).LengthSquared() }

// Distance returns the Euclidean distance between v and o.
func (v Vec4[T]) Distance(o Vec4[T]) float64 {
	x, y, z, w := v.floats()
	ox, oy, oz, ow := o.floats()
	return math.Hypot(math.Hypot(x-ox, y-oy), math.Hypot(z-oz, w-ow))
}

// scaled returns v with every component multiplied by k in floating point.
func (v Vec4[T]) scaled(k float64) Vec4[T] {
	return v.each(func(a T) T { return T(float64(a) * k) })
}

// Normalize returns the unit vector pointing in the same direction as v. The zero
// vector has no direction, so it is returned unchanged.
func (v Vec4[T]) Normalize() Vec4[T] {
	l := v.Length()
	if l == 0 {
		return v
	}
	return v.each(func(a T) T { return T(float64(a) / l) })
}

// ClampLength returns v shortened to length limit if it is longer, keeping its
// direction, and v itself otherwise.
func (v Vec4[T]) ClampLength(limit float64) Vec4[T] {
	k, ok := clampScale(v.Length(), limit)
	if !ok {
		return v
	}
	return v.scaled(k)
}

// Lerp returns the linear interpolation between v (at t = 0) and o (at t = 1).
// Values of t outside [0, 1] extrapolate.
func (v Vec4[T]) Lerp(o Vec4[T], t float64) Vec4[T] {
	return v.with(o, func(a, b T) T { return lerp(a, b, t) })
}

// dotFloat returns the dot product of v and o in floating point.
func (v Vec4[T]) dotFloat(o Vec4[T]) float64 {
	x, y, z, w := v.floats()
	ox, oy, oz, ow := o.floats()
	return x*ox + y*oy + z*oz + w*ow
}

// Slerp returns the spherical interpolation between v (at t = 0) and o (at t = 1): the
// direction turns at a constant rate along the arc between them, and the length changes
// linearly. When the arc is not defined, because either vector is zero or they point
// the same or opposite ways, Slerp returns Lerp.
func (v Vec4[T]) Slerp(o Vec4[T], t float64) Vec4[T] {
	ka, kb, ok := slerpWeights(v.dotFloat(o), v.Length(), o.Length(), t)
	if !ok {
		return v.Lerp(o, t)
	}
	return v.with(o, func(a, b T) T { return T(ka*float64(a) + kb*float64(b)) })
}

// AngleTo returns the angle between v and o in radians, in [0, π]. It returns 0 if
// either vector is zero.
func (v Vec4[T]) AngleTo(o Vec4[T]) float64 {
	return angle(v.dotFloat(o), v.Length()*o.Length())
}

// Project returns the projection of v onto o: the component of v that points along
// o. Projecting onto the zero vector gives the zero vector.
func (v Vec4[T]) Project(o Vec4[T]) Vec4[T] {
	den := o.dotFloat(o)
	if den == 0 {
		return Vec4[T]{}
	}
	return o.scaled(v.dotFloat(o) / den)
}

// Reflect returns v reflected off a surface with normal n, v - 2(v·n)n. The normal
// must have unit length.
func (v Vec4[T]) Reflect(n Vec4[T]) Vec4[T] { return v.Sub(n.Scale(2 * v.Dot(n))) }

// Floor returns v with every component rounded down. Integer vectors are unchanged.
func (v Vec4[T]) Floor() Vec4[T] { return v.each(func(a T) T { return roundWith(a, math.Floor) }) }

// Ceil returns v with every component rounded up. Integer vectors are unchanged.
func (v Vec4[T]) Ceil() Vec4[T] { return v.each(func(a T) T { return roundWith(a, math.Ceil) }) }

// Round returns v with every component rounded to the nearest integer, halves away
// from zero. Integer vectors are unchanged.
func (v Vec4[T]) Round() Vec4[T] { return v.each(func(a T) T { return roundWith(a, math.Round) }) }

// IsZero reports whether v is the zero vector.
func (v Vec4[T]) IsZero() bool { return v == Vec4[T]{} }

// ApproxEqual reports whether every pair of components differs by at most eps.
func (v Vec4[T]) ApproxEqual(o Vec4[T], eps float64) bool {
	return approx(v.X, o.X, eps) && approx(v.Y, o.Y, eps) && approx(v.Z, o.Z, eps) && approx(v.W, o.W, eps)
}

// XYZ returns the 3D vector (X, Y, Z), dropping W.
func (v Vec4[T]) XYZ() Vec3[T] { return Vec3[T]{v.X, v.Y, v.Z} }

// ToN returns v as a VecN.
func (v Vec4[T]) ToN() VecN[T] { return VecN[T]{v.X, v.Y, v.Z, v.W} }

// String formats v as "(x, y, z, w)".
func (v Vec4[T]) String() string { return fmt.Sprintf("(%v, %v, %v, %v)", v.X, v.Y, v.Z, v.W) }
