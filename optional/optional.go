// Package optional provides Optional, a container for a value that may or may not be present.
//
// Optional is a small value type: creating, copying and querying one never allocates.
// The zero value is an empty Optional, so it can be declared and used without initialisation.
package optional

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Optional holds either a value of type T or nothing.
type Optional[T any] struct {
	value   T
	present bool
}

// Of returns an Optional containing v. Zero values such as 0 or "" count as present.
func Of[T any](v T) Optional[T] {
	return Optional[T]{value: v, present: true}
}

// OfNullable returns an Optional containing *p, or an empty Optional if p is nil.
// The pointed-to value is copied, so later changes made through p are not observed.
func OfNullable[T any](p *T) Optional[T] {
	if p == nil {
		return Optional[T]{}
	}
	return Of(*p)
}

// OfOk builds an Optional from Go's "comma ok" idiom, for example:
//
//	home := optional.OfOk(os.LookupEnv("HOME"))
func OfOk[T any](v T, ok bool) Optional[T] {
	if !ok {
		return Optional[T]{}
	}
	return Of(v)
}

// Empty returns an Optional with no value. It is equivalent to the zero value.
func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

// IsPresent reports whether a value is present.
func (o Optional[T]) IsPresent() bool {
	return o.present
}

// IsEmpty reports whether no value is present.
func (o Optional[T]) IsEmpty() bool {
	return !o.present
}

// Get returns the contained value, or the zero value of T if the Optional is empty.
func (o Optional[T]) Get() T {
	return o.value
}

// Value returns the contained value and whether it is present, mirroring Go's "comma ok" idiom:
//
//	if v, ok := opt.Value(); ok { ... }
func (o Optional[T]) Value() (T, bool) {
	return o.value, o.present
}

// MustGet returns the contained value and panics if the Optional is empty.
func (o Optional[T]) MustGet() T {
	if !o.present {
		panic("optional: MustGet called on an empty Optional")
	}
	return o.value
}

// OrElse returns the contained value, or other if the Optional is empty.
func (o Optional[T]) OrElse(other T) T {
	if o.present {
		return o.value
	}
	return other
}

// OrElseGet returns the contained value, or the result of supplier if the Optional is empty.
// supplier is only called when needed.
func (o Optional[T]) OrElseGet(supplier func() T) T {
	if o.present {
		return o.value
	}
	return supplier()
}

// Or returns o if it holds a value, otherwise the Optional produced by supplier.
func (o Optional[T]) Or(supplier func() Optional[T]) Optional[T] {
	if o.present {
		return o
	}
	return supplier()
}

// IfPresent calls f with the contained value if one is present.
func (o Optional[T]) IfPresent(f func(T)) {
	if o.present {
		f(o.value)
	}
}

// IfPresentOrElse calls f with the contained value if one is present, otherwise it calls orElse.
func (o Optional[T]) IfPresentOrElse(f func(T), orElse func()) {
	if o.present {
		f(o.value)
	} else {
		orElse()
	}
}

// Filter returns o if it holds a value that satisfies predicate, otherwise an empty Optional.
func (o Optional[T]) Filter(predicate func(T) bool) Optional[T] {
	if o.present && predicate(o.value) {
		return o
	}
	return Optional[T]{}
}

// Ptr returns a pointer to a copy of the contained value, or nil if the Optional is empty.
// It is the inverse of OfNullable.
func (o Optional[T]) Ptr() *T {
	if !o.present {
		return nil
	}
	v := o.value
	return &v
}

// IsZero reports whether the Optional is empty. It lets encoding/json drop empty
// Optionals from output when a field is tagged with `json:",omitzero"`.
func (o Optional[T]) IsZero() bool {
	return !o.present
}

func (o Optional[T]) String() string {
	if o.present {
		return fmt.Sprintf("Optional[%v]", o.value)
	}
	return "Optional.empty"
}

// MarshalJSON encodes the contained value, or null if the Optional is empty.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.present {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON decodes null as an empty Optional and anything else as a present value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*o = Optional[T]{}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = Of(v)
	return nil
}

// Map applies f to the contained value and wraps the result, or returns an empty
// Optional if o is empty. It is a function rather than a method because Go methods
// cannot introduce new type parameters.
func Map[T, R any](o Optional[T], f func(T) R) Optional[R] {
	if !o.present {
		return Optional[R]{}
	}
	return Of(f(o.value))
}

// FlatMap applies f to the contained value and returns its result, or returns an
// empty Optional if o is empty.
func FlatMap[T, R any](o Optional[T], f func(T) Optional[R]) Optional[R] {
	if !o.present {
		return Optional[R]{}
	}
	return f(o.value)
}
