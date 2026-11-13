package future

import (
	"context"
	"sync"

	"github.com/t1sk3/go-utils/types/result"
)

// GoCtx runs f on a new goroutine with ctx and returns a Future of its result. It is Go
// for a function that takes a context, which is what lets the work itself be cancelled:
//
//	user := future.GoCtx(ctx, func(ctx context.Context) (User, error) { return fetchUser(ctx, id) })
//
// If ctx has already ended, f does not run and the Future holds the context's error.
// Otherwise the Future completes when f returns, so f must watch ctx for cancelling to
// have any effect. GoCtx panics if f is nil.
func GoCtx[T any](ctx context.Context, f func(context.Context) (T, error)) *Future[T] {
	if f == nil {
		panic("future: GoCtx requires a non-nil function")
	}
	if err := ctx.Err(); err != nil {
		return Failed[T](err)
	}
	return Go(func() (T, error) { return f(ctx) })
}

// IsDone reports whether the result is available, without waiting.
func (f *Future[T]) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// TryGet returns the result if it is available, without waiting. The boolean is false
// while the Future is still running.
func (f *Future[T]) TryGet() (result.Result[T], bool) {
	if !f.IsDone() {
		return result.Result[T]{}, false
	}
	return result.Of(f.value, f.err), true
}

// Recover returns a Future that holds the value of f, or, if f fails, the result of fn
// applied to its error. fn runs on a new goroutine once f has failed, and can return a
// substitute value or another error:
//
//	page := future.Go(fetchLive).Recover(func(error) (Page, error) { return loadCached() })
//
// Recover panics if fn is nil.
func (f *Future[T]) Recover(fn func(error) (T, error)) *Future[T] {
	if fn == nil {
		panic("future: Recover requires a non-nil function")
	}
	return Go(func() (T, error) {
		<-f.done
		if f.err != nil {
			return fn(f.err)
		}
		return f.value, nil
	})
}

// OrElse returns a Future that holds the value of f, or fallback if f fails.
func (f *Future[T]) OrElse(fallback T) *Future[T] {
	return f.Recover(func(error) (T, error) { return fallback, nil })
}

// AllSettled waits for every future and returns each outcome, in the order given,
// whether it succeeded or failed. Unlike All it does not stop at the first failure, so
// use it when each result matters on its own. The error is only ever the context's: if
// ctx ends before every future has finished, AllSettled returns it and no results,
// unless they were all available already. AllSettled panics if a future is nil.
func AllSettled[T any](ctx context.Context, futures ...*Future[T]) ([]result.Result[T], error) {
	checkFutures(futures, "AllSettled")
	results := make([]result.Result[T], len(futures))
	// Every future has to finish, so waiting for them one after another takes no longer
	// than waiting for them all at once.
	for i, fut := range futures {
		if !fut.IsDone() {
			select {
			case <-fut.done:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		results[i] = result.Of(fut.value, fut.err)
	}
	return results, nil
}

// Promise is a Future that is completed by hand instead of by a function. It bridges
// code that reports its result some other way, such as a callback or a message from
// another system, to code that waits on a Future:
//
//	p := future.NewPromise[Reply]()
//	client.OnReply(id, func(r Reply) { p.Complete(r) })
//	reply, err := p.Future().Get(ctx)
//
// A Promise is completed once: the first call to Complete or Fail wins and later ones
// do nothing. It is safe for concurrent use and must be created with NewPromise.
type Promise[T any] struct {
	fut  *Future[T]
	once sync.Once
}

// NewPromise returns a Promise that has not been completed.
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{fut: &Future[T]{done: make(chan struct{})}}
}

// Future returns the Future that p completes. Every call returns the same one.
func (p *Promise[T]) Future() *Future[T] {
	return p.fut
}

// Complete makes the Future hold value. It reports whether this call completed the
// Promise, which is false if it was completed before.
func (p *Promise[T]) Complete(value T) bool {
	return p.settle(value, nil)
}

// Fail makes the Future hold err. It reports whether this call completed the Promise,
// which is false if it was completed before. Fail panics if err is nil.
func (p *Promise[T]) Fail(err error) bool {
	if err == nil {
		panic("future: Fail requires a non-nil error")
	}
	var zero T
	return p.settle(zero, err)
}

func (p *Promise[T]) settle(value T, err error) bool {
	won := false
	p.once.Do(func() {
		p.fut.value, p.fut.err = value, err
		close(p.fut.done)
		won = true
	})
	return won
}

// Pool runs functions on goroutines of their own while letting only a fixed number of
// them work at once. It is how to start many tasks without overwhelming whatever they
// use, such as a database or a remote API:
//
//	pool := future.NewPool(8)
//	for _, url := range urls {
//		pages = append(pages, pool.Go(func() (Page, error) { return fetch(url) }))
//	}
//	results, err := future.All(ctx, pages...)
//
// Go never blocks. A task beyond the limit waits on a goroutine of its own, which is
// cheap, so the limit is on work in progress and not on how many tasks can be queued. A
// Pool is safe for concurrent use and must be created with NewPool.
type Pool struct {
	slots chan struct{}
	tasks sync.WaitGroup
}

// NewPool returns a Pool that lets n functions run at once. It panics if n is not
// positive.
func NewPool(n int) *Pool {
	if n <= 0 {
		panic("future: NewPool requires a positive limit")
	}
	return &Pool{slots: make(chan struct{}, n)}
}

// Go runs f once one of the Pool's slots is free and returns a Future of its result
// straight away. As with the Go function, a panic in f becomes a *PanicError. Go panics
// if f is nil.
func (p *Pool) Go[T any](f func() (T, error)) *Future[T] {
	if f == nil {
		panic("future: Pool.Go requires a non-nil function")
	}
	p.tasks.Add(1)
	// The task counts as finished only after its Future has completed, so that every
	// Future is done by the time Wait returns.
	return start(func() (T, error) {
		p.slots <- struct{}{}
		defer func() { <-p.slots }()
		return f()
	}, p.tasks.Done)
}

// Wait blocks until every function passed to Go so far has returned and its Future has
// completed. It must not be called from inside one of them.
func (p *Pool) Wait() {
	p.tasks.Wait()
}
