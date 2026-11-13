package stream

import (
	"context"
	"iter"
	"sync"
)

// ParallelStream is a Stream whose stages run on several goroutines. Create one with
// Stream.Parallel, add stages with Filter, Map or Peek, and return to ordinary
// one-at-a-time processing with Sequential:
//
//	thumbs := stream.FromSlice(images).Parallel(8).
//		Map(resize).
//		Filter(isValid).
//		Sequential().
//		ToSlice()
//
// Only the stages between Parallel and Sequential run concurrently. Stages before
// Parallel run on a single goroutine that reads the source, so the source does not need
// to be safe for concurrent use. Stages after Sequential run on the caller's goroutine.
//
// Results always come out in the order of the source, so a slow element holds back the
// results behind it. At most n stage calls run at once, and at most about n results wait
// to be consumed, so infinite sources are fine.
//
// Every element takes a goroutine and a channel, which costs on the order of a
// microsecond. That only pays off when the stages do real work, such as I/O or heavy
// computation. For cheap functions, a sequential Stream is faster; to spread a lot of
// cheap work across goroutines, pass batches from Chunk through the parallel stage.
//
// Functions passed to the stages must be safe to call concurrently. A panic in a stage
// or the source is raised again on the goroutine consuming the stream, in the position
// of the element that caused it, with the original value but not the original stack.
// When a consumer stops early, such as with Limit or FindFirst, stages may already have
// run for up to about n elements past that point, so they should not have side effects
// that depend on it. The terminal operation returns only after running calls finish, so
// no callback outlives it. If the source blocks, that wait includes the source's current
// read.
//
// WithContext lets a context cut the stream short, and Unordered trades the ordering
// for throughput.
//
// The zero value is an empty ParallelStream.
type ParallelStream[T any] struct {
	// tasks produces one function per source element, which runs the parallel stages
	// and reports whether the element survived them.
	tasks iter.Seq[func() (T, bool)]
	n     int
	size  int
	// ctx, when not nil, stops the stream early once it ends.
	ctx context.Context
	// unordered makes results come out as they finish instead of in source order.
	unordered bool
}

// WithContext returns a ParallelStream that stops early when ctx ends. From that moment
// no further element is read from the source and no further stage call starts; the
// calls already running finish, and their results still come out. The stream then ends
// as if the source had, so check ctx.Err() afterwards to tell a complete run from one
// that was cut short.
//
//	pages := urls.Parallel(8).WithContext(ctx).Map(fetch).ToSlice()
//	if err := ctx.Err(); err != nil { ... } // pages holds only the first part
//
// The stages are not interrupted: to stop a call that is already running, have the
// stage function watch ctx itself.
func (p ParallelStream[T]) WithContext(ctx context.Context) ParallelStream[T] {
	p.ctx = ctx
	// A stream that can be cut short has no known size.
	p.size = -1
	return p
}

// Unordered returns a ParallelStream whose results come out in the order they finish
// instead of the order of the source. A slow element then no longer holds back the
// results behind it, which gives better throughput when the elements take very
// different amounts of time and their order does not matter.
//
// A panic in a stage is still raised on the consuming goroutine, but among the results,
// not at the position of the element that caused it.
func (p ParallelStream[T]) Unordered() ParallelStream[T] {
	p.unordered = true
	return p
}

// Parallel returns a ParallelStream of the elements of s whose stages run on at most n
// goroutines at once. Use runtime.GOMAXPROCS(0) for CPU-bound work, and whatever limit
// the resource can take for I/O. Parallel panics if n is not positive.
func (s Stream[T]) Parallel(n int) ParallelStream[T] {
	if n <= 0 {
		panic("stream: Parallel requires a positive number of goroutines")
	}
	src := s.All()
	return ParallelStream[T]{n: n, size: s.size, tasks: func(yield func(func() (T, bool)) bool) {
		for v := range src {
			if !yield(func() (T, bool) { return v, true }) {
				return
			}
		}
	}}
}

// Filter returns a ParallelStream of the elements that satisfy predicate.
func (p ParallelStream[T]) Filter(predicate func(T) bool) ParallelStream[T] {
	checkFunc(predicate == nil, "Filter")
	return parallelStage(p, -1, func(v T) (T, bool) { return v, predicate(v) })
}

// Map returns a ParallelStream of the results of applying mapper to each element. The
// result may have a different element type.
func (p ParallelStream[T]) Map[R any](mapper func(T) R) ParallelStream[R] {
	checkFunc(mapper == nil, "Map")
	return parallelStage(p, p.size, func(v T) (R, bool) { return mapper(v), true })
}

// Peek returns a ParallelStream that calls action on each element as a parallel stage.
func (p ParallelStream[T]) Peek(action func(T)) ParallelStream[T] {
	checkFunc(action == nil, "Peek")
	return parallelStage(p, p.size, func(v T) (T, bool) {
		action(v)
		return v, true
	})
}

// ForEach calls action on each element concurrently, in no particular order, and
// returns when every call has finished. To handle elements one at a time in order, use
// Sequential().ForEach.
func (p ParallelStream[T]) ForEach(action func(T)) {
	checkFunc(action == nil, "ForEach")
	for range p.Peek(action).Sequential().All() {
	}
}

// ToSlice collects the elements in order into a new slice. The result is never nil.
func (p ParallelStream[T]) ToSlice() []T {
	return p.Sequential().ToSlice()
}

// Sequential returns a Stream of the elements of p in source order. The parallel stages
// start when a terminal operation runs, once for each run.
func (p ParallelStream[T]) Sequential() Stream[T] {
	if p.tasks == nil {
		return Stream[T]{}
	}
	return Stream[T]{size: p.size, seq: func(yield func(T) bool) {
		var ctxDone <-chan struct{}
		if p.ctx != nil {
			ctxDone = p.ctx.Done()
		}
		if p.unordered {
			runUnordered(p.tasks, p.n, ctxDone, yield)
		} else {
			runParallel(p.tasks, p.n, ctxDone, yield)
		}
	}}
}

func parallelStage[T, R any](p ParallelStream[T], size int, stage func(T) (R, bool)) ParallelStream[R] {
	if p.tasks == nil {
		return ParallelStream[R]{}
	}
	tasks := p.tasks
	if p.ctx != nil {
		size = -1
	}
	return ParallelStream[R]{n: p.n, size: size, ctx: p.ctx, unordered: p.unordered, tasks: func(yield func(func() (R, bool)) bool) {
		for task := range tasks {
			next := func() (R, bool) {
				if v, ok := task(); ok {
					return stage(v)
				}
				var zero R
				return zero, false
			}
			if !yield(next) {
				return
			}
		}
	}}
}

// parallelResult is the outcome of one element.
type parallelResult[T any] struct {
	value      T
	ok         bool
	panicked   bool
	panicValue any
}

// runParallel reads tasks on one goroutine, runs each on its own goroutine with at most
// n running at once, and passes the results to yield in the order of tasks. It returns
// only after every goroutine it started has finished. When ctxDone is closed, it stops
// reading tasks; ctxDone may be nil.
func runParallel[T any](tasks iter.Seq[func() (T, bool)], n int, ctxDone <-chan struct{}, yield func(T) bool) {
	// done tells the reader that the consumer has stopped.
	done := make(chan struct{})
	// pending holds a result slot per element, in order. Its capacity limits how far the
	// reader can get ahead of the consumer.
	pending := make(chan chan parallelResult[T], n)
	// running limits how many tasks execute at once.
	running := make(chan struct{}, n)

	var wg sync.WaitGroup
	wg.Go(func() {
		defer close(pending)
		finished := false
		defer func() {
			if finished {
				return
			}
			// The source panicked. Report it after the results before it.
			slot := make(chan parallelResult[T], 1)
			slot <- failedResult[T](recover())
			select {
			case pending <- slot:
			case <-done:
			}
		}()
		for task := range tasks {
			// Checked on its own first: a select with several ready cases picks one at
			// random, and an ended context has to win.
			select {
			case <-ctxDone:
				finished = true
				return
			default:
			}
			slot := make(chan parallelResult[T], 1)
			select {
			case pending <- slot:
			case <-done:
				finished = true
				return
			case <-ctxDone:
				finished = true
				return
			}
			select {
			case running <- struct{}{}:
			case <-done:
				finished = true
				return
			case <-ctxDone:
				// The consumer is still reading and will wait for this slot, so fill it
				// with a result that says there is no element.
				slot <- parallelResult[T]{}
				finished = true
				return
			}
			wg.Go(func() {
				defer func() { <-running }()
				runTask(task, func(r parallelResult[T]) { slot <- r })
			})
		}
		finished = true
	})

	defer func() {
		close(done)
		wg.Wait()
	}()
	for slot := range pending {
		r := <-slot
		if r.panicked {
			panic(r.panicValue)
		}
		if r.ok && !yield(r.value) {
			return
		}
	}
}

// runTask runs one task and hands its result to deliver, turning a panic into a result.
// The result is delivered from a deferred call and not returned, because a task that
// calls runtime.Goexit never returns, while its deferred calls still run.
func runTask[T any](task func() (T, bool), deliver func(parallelResult[T])) {
	var r parallelResult[T]
	completed := false
	defer func() {
		if !completed {
			r = failedResult[T](recover())
		}
		deliver(r)
	}()
	r.value, r.ok = task()
	completed = true
}

// runUnordered is runParallel without the ordering: results are passed to yield as
// they finish. At most n tasks run at once, and at most about n results wait to be
// consumed, because a task keeps its place among the n until its result is taken.
func runUnordered[T any](tasks iter.Seq[func() (T, bool)], n int, ctxDone <-chan struct{}, yield func(T) bool) {
	done := make(chan struct{})
	results := make(chan parallelResult[T])
	running := make(chan struct{}, n)

	var reader sync.WaitGroup
	reader.Go(func() {
		var workers sync.WaitGroup
		finished := false
		defer func() {
			if !finished {
				// The source panicked.
				select {
				case results <- failedResult[T](recover()):
				case <-done:
				}
			}
			workers.Wait()
			close(results)
		}()
		for task := range tasks {
			select {
			case <-ctxDone:
				finished = true
				return
			default:
			}
			select {
			case running <- struct{}{}:
			case <-done:
				finished = true
				return
			case <-ctxDone:
				finished = true
				return
			}
			workers.Go(func() {
				defer func() { <-running }()
				runTask(task, func(r parallelResult[T]) {
					select {
					case results <- r:
					case <-done:
					}
				})
			})
		}
		finished = true
	})

	defer func() {
		close(done)
		reader.Wait()
	}()
	for r := range results {
		if r.panicked {
			panic(r.panicValue)
		}
		if r.ok && !yield(r.value) {
			return
		}
	}
}

// failedResult builds the result of a goroutine that did not complete. The recovered
// value is nil only if the function called runtime.Goexit, as t.FailNow does.
func failedResult[T any](recovered any) parallelResult[T] {
	if recovered == nil {
		recovered = "stream: a parallel stage or its source called runtime.Goexit"
	}
	return parallelResult[T]{panicked: true, panicValue: recovered}
}
