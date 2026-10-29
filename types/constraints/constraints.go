// Package constraints defines type constraints for generic code that works with
// numbers and other basic types. Every constraint uses ~ so that named types such
// as time.Duration or `type Celsius float64` satisfy it too.
//
// The constraints nest as follows:
//
//	Basic        = Numeric | ~string | ~bool
//	Numeric      = Number | Complex
//	Number       = Integer | Float          (real numbers, ordered)
//	Integer      = Signed | Unsigned
//	SignedNumber = Signed | Float | Complex (supports unary minus)
package constraints

// Signed is a constraint matching all signed integer types.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is a constraint matching all unsigned integer types.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer is a constraint matching all integer types.
type Integer interface {
	Signed | Unsigned
}

// Float is a constraint matching all floating-point types.
type Float interface {
	~float32 | ~float64
}

// Complex is a constraint matching all complex number types.
type Complex interface {
	~complex64 | ~complex128
}

// Number is a constraint matching all integer and floating-point types: the real
// numbers. Unlike Numeric it excludes complex types, so values can be ordered with
// < and >.
type Number interface {
	Integer | Float
}

// Numeric is a constraint matching every numeric type, including complex numbers.
// It permits arithmetic (+ - * /) and equality, but not ordering.
type Numeric interface {
	Number | Complex
}

// SignedNumber is a constraint matching every numeric type that can hold a
// negative value: signed integers, floats and complex numbers. It is the set of
// types for which unary minus is meaningful.
type SignedNumber interface {
	Signed | Float | Complex
}

// Basic is a constraint matching the numeric, string and boolean types.
//
// Every Basic type is comparable, but most generic code that only needs == should
// use the built-in comparable constraint, which also admits structs, arrays,
// pointers and other comparable types.
type Basic interface {
	Numeric | ~string | ~bool
}
