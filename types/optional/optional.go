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

// FromMap returns an Optional containing m[key], or an empty Optional if key is not in
// m. A key that is present with a zero value counts as present.
func FromMap[K comparable, V any](m map[K]V, key K) Optional[V] {
	v, ok := m[key]
	return OfOk(v, ok)
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
//
// Unlike Java's Optional.get, it does not fail on an empty Optional, so an absent
// value and a present zero value look the same. Use Value or IsPresent to tell them
// apart, MustGet to panic when empty, and OrZero where the zero value is what you
// want and the call should say so.
func (o Optional[T]) Get() T {
	return o.value
}

// OrZero returns the contained value, or the zero value of T if the Optional is empty.
// It is the same as Get under a name that states what happens when empty.
func (o Optional[T]) OrZero() T {
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

// OrElseErr returns the contained value and a nil error, or the zero value of T and err
// if the Optional is empty. It is the error-returning form of Java's orElseThrow:
//
//	u, err := findUser(id).OrElseErr(ErrNotFound)
func (o Optional[T]) OrElseErr(err error) (T, error) {
	if o.present {
		return o.value, nil
	}
	var zero T
	return zero, err
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
	return new(o.value)
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
// Optional if o is empty. The result may have a different type:
//
//	name := findUser(id).Map(User.Name).OrElse("anonymous")
func (o Optional[T]) Map[R any](f func(T) R) Optional[R] {
	if !o.present {
		return Optional[R]{}
	}
	return Of(f(o.value))
}

// FlatMap applies f to the contained value and returns its result, or returns an
// empty Optional if o is empty. Use it when f itself returns an Optional:
//
//	email := findUser(id).FlatMap(User.Email)
func (o Optional[T]) FlatMap[R any](f func(T) Optional[R]) Optional[R] {
	if !o.present {
		return Optional[R]{}
	}
	return f(o.value)
}

// Equal reports whether a and b are both empty, or both hold the same value. For
// comparable T it gives the same answer as a == b; it exists for use as a function
// value, such as with slices.EqualFunc.
func Equal[T comparable](a, b Optional[T]) bool {
	return a.present == b.present && a.value == b.value
}

// EqualFunc reports whether a and b are both empty, or both hold values that eq
// reports as equal. Use it when T is not comparable, such as a slice.
func EqualFunc[T any](a, b Optional[T], eq func(T, T) bool) bool {
	if !a.present || !b.present {
		return a.present == b.present
	}
	return eq(a.value, b.value)
}
