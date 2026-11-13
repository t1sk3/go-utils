// Package result provides Result, a value that is either a success or an error.
//
// A Result carries the two return values of a fallible function as one, so that the
// outcome can be stored in a slice, sent over a channel or passed through generic code
// that only has room for one type:
//
//	r := result.Of(strconv.Atoi(s))
//	n := r.Map(double).OrElse(0)
//
// Ordinary Go code should keep returning (T, error). Reach for Result where a single
// value is needed, as stream.TryMap and future.AllSettled do. The zero value is a
// successful Result holding the zero value of T.
package result

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/optional"
)

// Result holds either a value of type T or an error.
type Result[T any] struct {
	value T
	err   error
}

// Ok returns a successful Result holding v.
func Ok[T any](v T) Result[T] {
	return Result[T]{value: v}
}

// Err returns a failed Result holding err. It panics if err is nil, because a Result
// without an error is a success.
func Err[T any](err error) Result[T] {
	if err == nil {
		panic("result: Err requires a non-nil error")
	}
	return Result[T]{err: err}
}

// Of builds a Result from the two return values of a fallible function:
//
//	r := result.Of(os.ReadFile(path))
//
// The value is dropped when err is not nil.
func Of[T any](v T, err error) Result[T] {
	if err != nil {
		return Result[T]{err: err}
	}
	return Result[T]{value: v}
}

// IsOk reports whether r is a success.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr reports whether r is a failure.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Get returns the value and the error, turning r back into Go's usual two results. The
// value is the zero value of T when r is a failure.
func (r Result[T]) Get() (T, error) {
	return r.value, r.err
}

// Err returns the error, or nil if r is a success.
func (r Result[T]) Err() error {
	return r.err
}

// MustGet returns the value and panics with the error if r is a failure.
func (r Result[T]) MustGet() T {
	if r.err != nil {
		panic(fmt.Errorf("result: MustGet called on a failed Result: %w", r.err))
	}
	return r.value
}

// OrElse returns the value, or other if r is a failure.
func (r Result[T]) OrElse(other T) T {
	if r.err != nil {
		return other
	}
	return r.value
}

// OrElseGet returns the value, or the result of calling fallback with the error if r
// is a failure.
func (r Result[T]) OrElseGet(fallback func(error) T) T {
	if r.err != nil {
		return fallback(r.err)
	}
	return r.value
}

// Optional returns the value as an Optional, which is empty if r is a failure. The
// error is dropped.
func (r Result[T]) Optional() optional.Optional[T] {
	return optional.OfOk(r.value, r.err == nil)
}

// Map applies f to the value and wraps the result, or passes the error through if r is
// a failure. The result may have a different type.
func (r Result[T]) Map[R any](f func(T) R) Result[R] {
	if r.err != nil {
		return Result[R]{err: r.err}
	}
	return Result[R]{value: f(r.value)}
}

// FlatMap applies f, which can fail itself, to the value, or passes the error through
// if r is a failure:
//
//	port := result.Of(os.ReadFile(path)).FlatMap(parsePort)
func (r Result[T]) FlatMap[R any](f func(T) (R, error)) Result[R] {
	if r.err != nil {
		return Result[R]{err: r.err}
	}
	return Of(f(r.value))
}

// MapErr applies f to the error, for example to add context, or returns r unchanged if
// it is a success. If f returns nil, the Result becomes a success holding the zero
// value of T.
func (r Result[T]) MapErr(f func(error) error) Result[T] {
	if r.err == nil {
		return r
	}
	return Result[T]{err: f(r.err)}
}

// String formats r as "Ok(value)" or "Err(message)".
func (r Result[T]) String() string {
	if r.err != nil {
		return fmt.Sprintf("Err(%v)", r.err)
	}
	return fmt.Sprintf("Ok(%v)", r.value)
}
