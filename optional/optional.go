package utils

import "fmt"

// Optional is a generic interface that models a value that may or may not be present.
// IsPresent reports whether a value is present.
// Get returns the contained value; calling Get when no value is present is invalid
// and implementations may panic. Callers should check IsPresent before calling Get.
type Optional[E any] interface {
	IsPresent() bool
	Get() E
}

type optionalImpl[E any] struct {
	value *E
}

func (o *optionalImpl[E]) IsPresent() bool {
	return o.value != nil
}

func (o *optionalImpl[E]) Get() E {
	if o.value == nil {
		var zero E
		return zero
	}
	return *o.value
}

func Of[E any](value E) Optional[E] {
	return &optionalImpl[E]{value: &value}
}

func OfNullable[E any](value *E) Optional[E] {
	return &optionalImpl[E]{value: value}
}

func Empty[E any]() Optional[E] {
	return &optionalImpl[E]{value: nil}
}

func (o *optionalImpl[E]) String() string {
	if o.IsPresent() {
		return "Optional[" + fmt.Sprintf("%v", *o.value) + "]"
	}
	return "Optional.empty"
}
