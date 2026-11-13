package vector

import (
	"fmt"
	"math"

	"github.com/t1sk3/go-utils/types/constraints"
)

// Vec3 is a three-dimensional vector. It is a plain comparable struct: the zero value
// is the zero vector, == compares components, and it never allocates.
type Vec3[T constraints.Number] struct {
	X T `json:"x"`
	Y T `json:"y"`
	Z T `json:"z"`
}

// Of3 returns the vector (x, y, z).
func Of3[T constraints.Number](x, y, z T) Vec3[T] {
	return Vec3[T]{x, y, z}
}

// Convert3 returns v with each component converted to U.
func Convert3[U, T constraints.Number](v Vec3[T]) Vec3[U] {
	return Vec3[U]{U(v.X), U(v.Y), U(v.Z)}
}

// Dim returns 3.
func (v Vec3[T]) Dim() int { return 3 }

// Add returns v + o.
func (v Vec3[T]) Add(o Vec3[T]) Vec3[T] { return Vec3[T]{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// Sub returns v - o.
func (v Vec3[T]) Sub(o Vec3[T]) Vec3[T] { return Vec3[T]{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }

// Mul returns the component-wise (Hadamard) product of v and o.
func (v Vec3[T]) Mul(o Vec3[T]) Vec3[T] { return Vec3[T]{v.X * o.X, v.Y * o.Y, v.Z * o.Z} }

// Div returns the component-wise quotient of v and o.
func (v Vec3[T]) Div(o Vec3[T]) Vec3[T] { return Vec3[T]{v.X / o.X, v.Y / o.Y, v.Z / o.Z} }

// Scale returns v multiplied by the scalar s.
func (v Vec3[T]) Scale(s T) Vec3[T] { return Vec3[T]{v.X * s, v.Y * s, v.Z * s} }

// DivScalar returns v divided by the scalar s.
func (v Vec3[T]) DivScalar(s T) Vec3[T] { return Vec3[T]{v.X / s, v.Y / s, v.Z / s} }

// Neg returns -v. For unsigned T the components wrap around, as unary minus does.
func (v Vec3[T]) Neg() Vec3[T] { return Vec3[T]{-v.X, -v.Y, -v.Z} }

// Abs returns v with every component replaced by its absolute value. The smallest
// value of a signed integer type has no positive counterpart and stays as it is, as
// with unary minus.
func (v Vec3[T]) Abs() Vec3[T] { return Vec3[T]{abs(v.X), abs(v.Y), abs(v.Z)} }

// Min returns the component-wise minimum of v and o.
func (v Vec3[T]) Min(o Vec3[T]) Vec3[T] {
	return Vec3[T]{min(v.X, o.X), min(v.Y, o.Y), min(v.Z, o.Z)}
}

// Max returns the component-wise maximum of v and o.
func (v Vec3[T]) Max(o Vec3[T]) Vec3[T] {
	return Vec3[T]{max(v.X, o.X), max(v.Y, o.Y), max(v.Z, o.Z)}
}

// Clamp returns v with every component limited to the range given by the matching
// components of lo and hi.
func (v Vec3[T]) Clamp(lo, hi Vec3[T]) Vec3[T] { return v.Max(lo).Min(hi) }

// Sum returns X + Y + Z.
func (v Vec3[T]) Sum() T { return v.X + v.Y + v.Z }

// Dot returns the dot product of v and o.
func (v Vec3[T]) Dot(o Vec3[T]) T { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// Cross returns the cross product v × o: a vector perpendicular to both, following
// the right-hand rule, whose length is the area of the parallelogram they span.
func (v Vec3[T]) Cross(o Vec3[T]) Vec3[T] {
	return Vec3[T]{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

// LengthSquared returns the squared Euclidean length of v, v·v. It avoids the square
// root of Length and stays in T, so it suits comparing lengths.
func (v Vec3[T]) LengthSquared() T { return v.Dot(v) }

// Length returns the Euclidean length (magnitude) of v.
func (v Vec3[T]) Length() float64 {
	return hypot3(float64(v.X), float64(v.Y), float64(v.Z))
}

// DistanceSquared returns the squared Euclidean distance between v and o.
func (v Vec3[T]) DistanceSquared(o Vec3[T]) T { return v.Sub(o).LengthSquared() }

// Distance returns the Euclidean distance between v and o.
func (v Vec3[T]) Distance(o Vec3[T]) float64 {
	x := float64(v.X) - float64(o.X)
	y := float64(v.Y) - float64(o.Y)
	z := float64(v.Z) - float64(o.Z)
	return hypot3(x, y, z)
}

// Normalize returns the unit vector pointing in the same direction as v. The zero
// vector has no direction, so it is returned unchanged.
func (v Vec3[T]) Normalize() Vec3[T] {
	l := v.Length()
	if l == 0 {
		return v
	}
	return Vec3[T]{T(float64(v.X) / l), T(float64(v.Y) / l), T(float64(v.Z) / l)}
}

// Lerp returns the linear interpolation between v (at t = 0) and o (at t = 1).
// Values of t outside [0, 1] extrapolate.
func (v Vec3[T]) Lerp(o Vec3[T], t float64) Vec3[T] {
	return Vec3[T]{lerp(v.X, o.X, t), lerp(v.Y, o.Y, t), lerp(v.Z, o.Z, t)}
}

// Slerp returns the spherical interpolation between v (at t = 0) and o (at t = 1): the
// direction turns at a constant rate along the arc between them, and the length changes
// linearly. When the arc is not defined, because either vector is zero or they point
// the same or opposite ways, Slerp returns Lerp.
func (v Vec3[T]) Slerp(o Vec3[T], t float64) Vec3[T] {
	dot := float64(v.X)*float64(o.X) + float64(v.Y)*float64(o.Y) + float64(v.Z)*float64(o.Z)
	ka, kb, ok := slerpWeights(dot, v.Length(), o.Length(), t)
	if !ok {
		return v.Lerp(o, t)
	}
	return Vec3[T]{
		T(ka*float64(v.X) + kb*float64(o.X)),
		T(ka*float64(v.Y) + kb*float64(o.Y)),
		T(ka*float64(v.Z) + kb*float64(o.Z)),
	}
}

// Rotate returns v rotated by rad radians around axis, counter-clockwise when looking
// from the tip of axis towards the origin (the right-hand rule). The axis does not
// need unit length. Rotating around the zero vector returns v unchanged.
func (v Vec3[T]) Rotate(axis Vec3[T], rad float64) Vec3[T] {
	l := axis.Length()
	if l == 0 {
		return v
	}
	// Rodrigues' formula: v cos + (k × v) sin + k (k · v)(1 - cos), with k the unit axis.
	kx, ky, kz := float64(axis.X)/l, float64(axis.Y)/l, float64(axis.Z)/l
	x, y, z := float64(v.X), float64(v.Y), float64(v.Z)
	sin, cos := math.Sincos(rad)
	along := (kx*x + ky*y + kz*z) * (1 - cos)
	return Vec3[T]{
		T(x*cos + (ky*z-kz*y)*sin + kx*along),
		T(y*cos + (kz*x-kx*z)*sin + ky*along),
		T(z*cos + (kx*y-ky*x)*sin + kz*along),
	}
}

// ClampLength returns v shortened to length limit if it is longer, keeping its
// direction, and v itself otherwise.
func (v Vec3[T]) ClampLength(limit float64) Vec3[T] {
	k, ok := clampScale(v.Length(), limit)
	if !ok {
		return v
	}
	return Vec3[T]{T(float64(v.X) * k), T(float64(v.Y) * k), T(float64(v.Z) * k)}
}

// Floor returns v with every component rounded down. Integer vectors are unchanged.
func (v Vec3[T]) Floor() Vec3[T] {
	return Vec3[T]{roundWith(v.X, math.Floor), roundWith(v.Y, math.Floor), roundWith(v.Z, math.Floor)}
}

// Ceil returns v with every component rounded up. Integer vectors are unchanged.
func (v Vec3[T]) Ceil() Vec3[T] {
	return Vec3[T]{roundWith(v.X, math.Ceil), roundWith(v.Y, math.Ceil), roundWith(v.Z, math.Ceil)}
}

// Round returns v with every component rounded to the nearest integer, halves away
// from zero. Integer vectors are unchanged.
func (v Vec3[T]) Round() Vec3[T] {
	return Vec3[T]{roundWith(v.X, math.Round), roundWith(v.Y, math.Round), roundWith(v.Z, math.Round)}
}

// Extend returns the 4D vector (X, Y, Z, w).
func (v Vec3[T]) Extend(w T) Vec4[T] { return Vec4[T]{v.X, v.Y, v.Z, w} }

// AngleTo returns the angle between v and o in radians, in [0, π]. It returns 0 if
// either vector is zero.
func (v Vec3[T]) AngleTo(o Vec3[T]) float64 {
	dot := float64(v.X)*float64(o.X) + float64(v.Y)*float64(o.Y) + float64(v.Z)*float64(o.Z)
	return angle(dot, v.Length()*o.Length())
}

// Project returns the projection of v onto o: the component of v that points along
// o. Projecting onto the zero vector gives the zero vector.
func (v Vec3[T]) Project(o Vec3[T]) Vec3[T] {
	ox, oy, oz := float64(o.X), float64(o.Y), float64(o.Z)
	den := ox*ox + oy*oy + oz*oz
	if den == 0 {
		return Vec3[T]{}
	}
	k := (float64(v.X)*ox + float64(v.Y)*oy + float64(v.Z)*oz) / den
	return Vec3[T]{T(ox * k), T(oy * k), T(oz * k)}
}

// Reflect returns v reflected off a surface with normal n, v - 2(v·n)n. The normal
// must have unit length.
func (v Vec3[T]) Reflect(n Vec3[T]) Vec3[T] { return v.Sub(n.Scale(2 * v.Dot(n))) }

// IsZero reports whether v is the zero vector.
func (v Vec3[T]) IsZero() bool { return v == Vec3[T]{} }

// ApproxEqual reports whether every pair of components differs by at most eps.
func (v Vec3[T]) ApproxEqual(o Vec3[T], eps float64) bool {
	return approx(v.X, o.X, eps) && approx(v.Y, o.Y, eps) && approx(v.Z, o.Z, eps)
}

// XY returns the 2D vector (X, Y), dropping Z.
func (v Vec3[T]) XY() Vec2[T] { return Vec2[T]{v.X, v.Y} }

// ToN returns v as a VecN.
func (v Vec3[T]) ToN() VecN[T] { return VecN[T]{v.X, v.Y, v.Z} }

// String formats v as "(x, y, z)".
func (v Vec3[T]) String() string { return fmt.Sprintf("(%v, %v, %v)", v.X, v.Y, v.Z) }
