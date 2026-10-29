// Package constraints defines type constraints for generic code that works with
// numbers. Every constraint uses ~ so that named types such as time.Duration or
// `type Celsius float64` satisfy it too.
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

// Number is a constraint matching all integer and floating-point types.
type Number interface {
	Integer | Float
}
