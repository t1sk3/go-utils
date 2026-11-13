// Package future provides Future, the typed result of work running on another goroutine.
//
// It is a small counterpart of Java's CompletableFuture. It saves the channel, struct and
// error plumbing of getting one value back from a goroutine:
//
//	user := future.Go(func() (User, error) { return fetchUser(ctx, id) })
//	orders := future.Go(func() ([]Order, error) { return fetchOrders(ctx, id) })
//
//	u, err := user.Get(ctx)
//	...
//	o, err := orders.Get(ctx)
//
// Canceling the context passed to Get stops the wait, not the work. To cancel the work,
// have its function observe a context of its own, as above.
package future

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
)

// ErrNoFutures is returned by Any when it is called without any futures.
var ErrNoFutures = errors.New("future: Any called with no futures")

// PanicError is the error of a Future whose function panicked. Without it, the panic
// would crash the program from a goroutine the caller has no way to guard.
type PanicError struct {
	// Value is the value passed to panic.
	Value any
	// Stack is the stack trace of the goroutine at the time of the panic.
	Stack []byte
}

// Error describes the panic value. It leaves out the stack trace, because error
// strings tend to end up in logs and responses where source paths do not belong; read
// Stack to log it deliberately.
func (e *PanicError) Error() string {
	return fmt.Sprintf("future: function panicked: %v", e.Value)
}

// Unwrap returns the panic value if it is an error, so errors.Is and errors.As see it.
func (e *PanicError) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}

// Future is a value that a goroutine is computing. It is safe for concurrent use, and
// its result can be read any number of times.
type Future[T any] struct {
	done  chan struct{}
	value T
	err   error
}

// Go runs f on a new goroutine and returns a Future of its result. If f panics, the
// Future completes with a *PanicError. Go panics if f is nil.
func Go[T any](f func() (T, error)) *Future[T] {
	if f == nil {
		panic("future: Go requires a non-nil function")
	}
	fut := &Future[T]{done: make(chan struct{})}
	go func() {
		defer close(fut.done)
		completed := false
		defer func() {
			if !completed {
				fut.err = failure(recover())
			}
		}()
		fut.value, fut.err = f()
		completed = true
	}()
	return fut
}

// Completed returns a Future that already holds value.
func Completed[T any](value T) *Future[T] {
	fut := &Future[T]{done: make(chan struct{}), value: value}
	close(fut.done)
	return fut
}

// Failed returns a Future that already holds err.
func Failed[T any](err error) *Future[T] {
	fut := &Future[T]{done: make(chan struct{}), err: err}
	close(fut.done)
	return fut
}

// Then returns a Future of fn applied to the result of f, run on a new goroutine once f
// completes. If f fails, fn does not run and the returned Future holds the same error.
// The result may have a different type:
//
//	orders := future.Go(fetchUser).Then(fetchOrders)
//
// Then panics if fn is nil.
func (f *Future[T]) Then[R any](fn func(T) (R, error)) *Future[R] {
	if fn == nil {
		panic("future: Then requires a non-nil function")
	}
	return Go(func() (R, error) {
		<-f.done
		if f.err != nil {
			var zero R
			return zero, f.err
		}
		return fn(f.value)
	})
}

// Get waits for the result. If ctx ends first, it returns the zero value of T and the
// context's error. A result that is already available is returned even if ctx has ended.
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	default:
	}
	select {
	case <-f.done:
		return f.value, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// Done returns a channel that is closed when the result is available, for use in a
// select statement.
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// All waits for every future and returns their values in the order given. It returns
// as soon as any future fails or ctx ends, with that error, and the other futures keep
// running. An outcome that is already available is returned even if ctx has ended.
// With no futures, it returns an empty slice. All panics if a future is nil.
func All[T any](ctx context.Context, futures ...*Future[T]) ([]T, error) {
	checkFutures(futures, "All")
	// Results that are already available win over a context that has ended, as in Get.
	if values, err, ok := allReady(futures); ok {
		return values, err
	}
	failed := make(chan error, 1)
	finished := make(chan struct{})
	defer close(finished)
	remaining := make(chan struct{}, len(futures))
	for _, fut := range futures {
		go func() {
			select {
			case <-fut.done:
				if fut.err != nil {
					select {
					case failed <- fut.err:
					default:
					}
				}
				remaining <- struct{}{}
			case <-finished:
			}
		}()
	}
	for range futures {
		select {
		case err := <-failed:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-remaining:
		}
	}
	// The last future to finish may have failed just as its signal arrived.
	select {
	case err := <-failed:
		return nil, err
	default:
	}
	values, err, _ := allReady(futures)
	return values, err
}

// allReady returns the outcome of All without waiting, and false if some future that
// matters is still running. A failed future settles the outcome whatever the others
// are doing; the first one in argument order is reported.
func allReady[T any](futures []*Future[T]) (values []T, err error, ok bool) {
	pending := false
	for _, fut := range futures {
		select {
		case <-fut.done:
			if fut.err != nil {
				return nil, fut.err, true
			}
		default:
			pending = true
		}
	}
	if pending {
		return nil, nil, false
	}
	values = make([]T, len(futures))
	for i, fut := range futures {
		values[i] = fut.value
	}
	return values, nil, true
}

// checkFutures panics on the caller's goroutine for a nil Future. Without it, the nil
// would be dereferenced on a goroutine the caller cannot guard, crashing the program.
func checkFutures[T any](futures []*Future[T], op string) {
	for i, fut := range futures {
		if fut == nil {
			panic(fmt.Sprintf("future: %s called with a nil Future at index %d", op, i))
		}
	}
}

// Any returns the value of the first future to succeed, and the other futures keep
// running. If every future fails, it returns all their errors joined, in the order
// given. If ctx ends first, it returns the context's error, unless the outcome is
// already available. With no futures, it returns ErrNoFutures. Any panics if a future is nil.
func Any[T any](ctx context.Context, futures ...*Future[T]) (T, error) {
	var zero T
	if len(futures) == 0 {
		return zero, ErrNoFutures
	}
	checkFutures(futures, "Any")
	// An outcome that is already available wins over a context that has ended, as in
	// Get: the first value, or the errors if every future has failed.
	ready := make([]error, len(futures))
	allFailed := true
	for i, fut := range futures {
		select {
		case <-fut.done:
			if fut.err == nil {
				return fut.value, nil
			}
			ready[i] = fut.err
		default:
			allFailed = false
		}
	}
	if allFailed {
		return zero, errors.Join(ready...)
	}
	finished := make(chan struct{})
	defer close(finished)
	type outcome struct {
		index int
		value T
		err   error
	}
	outcomes := make(chan outcome, len(futures))
	for i, fut := range futures {
		go func() {
			select {
			case <-fut.done:
				outcomes <- outcome{i, fut.value, fut.err}
			case <-finished:
			}
		}()
	}
	errs := make([]error, len(futures))
	for range futures {
		select {
		case o := <-outcomes:
			if o.err == nil {
				return o.value, nil
			}
			errs[o.index] = o.err
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
	return zero, errors.Join(errs...)
}

// failure builds the error of a goroutine that did not complete. The recovered value
// is nil only if the function called runtime.Goexit, as t.FailNow does.
func failure(recovered any) error {
	if recovered == nil {
		recovered = "runtime.Goexit called"
	}
	return &PanicError{Value: recovered, Stack: debug.Stack()}
}
