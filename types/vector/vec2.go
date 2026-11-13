package vector

import (
	"fmt"
	"math"

	"github.com/t1sk3/go-utils/types/constraints"
)

// Vec2 is a two-dimensional vector. It is a plain comparable struct: the zero value
// is the zero vector, == compares components, and it never allocates.
type Vec2[T constraints.Number] struct {
	X T `json:"x"`
	Y T `json:"y"`
}

// Of2 returns the vector (x, y).
func Of2[T constraints.Number](x, y T) Vec2[T] {
	return Vec2[T]{x, y}
}

// FromPolar returns the vector with the given length that makes the angle rad (in
// radians, counter-clockwise from the positive X axis) with the X axis.
func FromPolar[T constraints.Number](length, rad float64) Vec2[T] {
	s, c := math.Sincos(rad)
	return Vec2[T]{T(length * c), T(length * s)}
}

// Convert2 returns v with each component converted to U.
func Convert2[U, T constraints.Number](v Vec2[T]) Vec2[U] {
	return Vec2[U]{U(v.X), U(v.Y)}
}

// Dim returns 2.
func (v Vec2[T]) Dim() int { return 2 }

// Add returns v + o.
func (v Vec2[T]) Add(o Vec2[T]) Vec2[T] { return Vec2[T]{v.X + o.X, v.Y + o.Y} }

// Sub returns v - o.
func (v Vec2[T]) Sub(o Vec2[T]) Vec2[T] { return Vec2[T]{v.X - o.X, v.Y - o.Y} }

// Mul returns the component-wise (Hadamard) product of v and o.
func (v Vec2[T]) Mul(o Vec2[T]) Vec2[T] { return Vec2[T]{v.X * o.X, v.Y * o.Y} }

// Div returns the component-wise quotient of v and o.
func (v Vec2[T]) Div(o Vec2[T]) Vec2[T] { return Vec2[T]{v.X / o.X, v.Y / o.Y} }

// Scale returns v multiplied by the scalar s.
func (v Vec2[T]) Scale(s T) Vec2[T] { return Vec2[T]{v.X * s, v.Y * s} }

// DivScalar returns v divided by the scalar s.
func (v Vec2[T]) DivScalar(s T) Vec2[T] { return Vec2[T]{v.X / s, v.Y / s} }

// Neg returns -v. For unsigned T the components wrap around, as unary minus does.
func (v Vec2[T]) Neg() Vec2[T] { return Vec2[T]{-v.X, -v.Y} }

// Abs returns v with every component replaced by its absolute value. The smallest
// value of a signed integer type has no positive counterpart and stays as it is, as
// with unary minus.
func (v Vec2[T]) Abs() Vec2[T] { return Vec2[T]{abs(v.X), abs(v.Y)} }

// Min returns the component-wise minimum of v and o.
func (v Vec2[T]) Min(o Vec2[T]) Vec2[T] { return Vec2[T]{min(v.X, o.X), min(v.Y, o.Y)} }

// Max returns the component-wise maximum of v and o.
func (v Vec2[T]) Max(o Vec2[T]) Vec2[T] { return Vec2[T]{max(v.X, o.X), max(v.Y, o.Y)} }

// Clamp returns v with every component limited to the range given by the matching
// components of lo and hi.
func (v Vec2[T]) Clamp(lo, hi Vec2[T]) Vec2[T] { return v.Max(lo).Min(hi) }

// Sum returns X + Y.
func (v Vec2[T]) Sum() T { return v.X + v.Y }

// Dot returns the dot product of v and o.
func (v Vec2[T]) Dot(o Vec2[T]) T { return v.X*o.X + v.Y*o.Y }

// Cross returns the Z component of the 3D cross product of v and o, v.X*o.Y - v.Y*o.X.
// It is positive when o is counter-clockwise from v, negative when clockwise and zero
// when they are parallel, and its magnitude is the area of the parallelogram they span.
func (v Vec2[T]) Cross(o Vec2[T]) T { return v.X*o.Y - v.Y*o.X }

// Perp returns v rotated 90° counter-clockwise, (-Y, X).
func (v Vec2[T]) Perp() Vec2[T] { return Vec2[T]{-v.Y, v.X} }

// Rotate returns v rotated counter-clockwise by rad radians.
func (v Vec2[T]) Rotate(rad float64) Vec2[T] {
	s, c := math.Sincos(rad)
	x, y := float64(v.X), float64(v.Y)
	return Vec2[T]{T(x*c - y*s), T(x*s + y*c)}
}

// Angle returns the angle of v in radians, counter-clockwise from the positive X axis,
// in (-π, π]. It is the inverse of FromPolar.
func (v Vec2[T]) Angle() float64 { return math.Atan2(float64(v.Y), float64(v.X)) }

// LengthSquared returns the squared Euclidean length of v, v·v. It avoids the square
// root of Length and stays in T, so it suits comparing lengths.
func (v Vec2[T]) LengthSquared() T { return v.Dot(v) }

// Length returns the Euclidean length (magnitude) of v.
func (v Vec2[T]) Length() float64 { return math.Hypot(float64(v.X), float64(v.Y)) }

// DistanceSquared returns the squared Euclidean distance between v and o.
func (v Vec2[T]) DistanceSquared(o Vec2[T]) T { return v.Sub(o).LengthSquared() }

// Distance returns the Euclidean distance between v and o.
func (v Vec2[T]) Distance(o Vec2[T]) float64 {
	return math.Hypot(float64(v.X)-float64(o.X), float64(v.Y)-float64(o.Y))
}

// Normalize returns the unit vector pointing in the same direction as v. The zero
// vector has no direction, so it is returned unchanged.
func (v Vec2[T]) Normalize() Vec2[T] {
	l := v.Length()
	if l == 0 {
		return v
	}
	return Vec2[T]{T(float64(v.X) / l), T(float64(v.Y) / l)}
}

// Lerp returns the linear interpolation between v (at t = 0) and o (at t = 1).
// Values of t outside [0, 1] extrapolate.
func (v Vec2[T]) Lerp(o Vec2[T], t float64) Vec2[T] {
	return Vec2[T]{lerp(v.X, o.X, t), lerp(v.Y, o.Y, t)}
}

// AngleTo returns the unsigned angle between v and o in radians, in [0, π]. It
// returns 0 if either vector is zero. Use Cross for the direction of rotation.
func (v Vec2[T]) AngleTo(o Vec2[T]) float64 {
	dot := float64(v.X)*float64(o.X) + float64(v.Y)*float64(o.Y)
	return angle(dot, v.Length()*o.Length())
}

// Project returns the projection of v onto o: the component of v that points along
// o. Projecting onto the zero vector gives the zero vector.
func (v Vec2[T]) Project(o Vec2[T]) Vec2[T] {
	ox, oy := float64(o.X), float64(o.Y)
	den := ox*ox + oy*oy
	if den == 0 {
		return Vec2[T]{}
	}
	k := (float64(v.X)*ox + float64(v.Y)*oy) / den
	return Vec2[T]{T(ox * k), T(oy * k)}
}

// Reflect returns v reflected off a surface with normal n, v - 2(v·n)n. The normal
// must have unit length.
func (v Vec2[T]) Reflect(n Vec2[T]) Vec2[T] { return v.Sub(n.Scale(2 * v.Dot(n))) }

// IsZero reports whether v is the zero vector.
func (v Vec2[T]) IsZero() bool { return v == Vec2[T]{} }

// ApproxEqual reports whether every pair of components differs by at most eps.
func (v Vec2[T]) ApproxEqual(o Vec2[T], eps float64) bool {
	return approx(v.X, o.X, eps) && approx(v.Y, o.Y, eps)
}

// Extend returns the 3D vector (X, Y, z).
func (v Vec2[T]) Extend(z T) Vec3[T] { return Vec3[T]{v.X, v.Y, z} }

// ToN returns v as a VecN.
func (v Vec2[T]) ToN() VecN[T] { return VecN[T]{v.X, v.Y} }

// String formats v as "(x, y)".
func (v Vec2[T]) String() string { return fmt.Sprintf("(%v, %v)", v.X, v.Y) }
