package stream

import (
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
// The zero value is an empty ParallelStream.
type ParallelStream[T any] struct {
	// tasks produces one function per source element, which runs the parallel stages
	// and reports whether the element survived them.
	tasks iter.Seq[func() (T, bool)]
	n     int
	size  int
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
	tasks, n := p.tasks, p.n
	return Stream[T]{size: p.size, seq: func(yield func(T) bool) {
		runParallel(tasks, n, yield)
	}}
}

func parallelStage[T, R any](p ParallelStream[T], size int, stage func(T) (R, bool)) ParallelStream[R] {
	if p.tasks == nil {
		return ParallelStream[R]{}
	}
	tasks := p.tasks
	return ParallelStream[R]{n: p.n, size: size, tasks: func(yield func(func() (R, bool)) bool) {
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
// only after every goroutine it started has finished.
func runParallel[T any](tasks iter.Seq[func() (T, bool)], n int, yield func(T) bool) {
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
			slot := make(chan parallelResult[T], 1)
			select {
			case pending <- slot:
			case <-done:
				finished = true
				return
			}
			select {
			case running <- struct{}{}:
			case <-done:
				finished = true
				return
			}
			wg.Go(func() {
				defer func() { <-running }()
				var r parallelResult[T]
				completed := false
				defer func() {
					if !completed {
						r = failedResult[T](recover())
					}
					slot <- r
				}()
				r.value, r.ok = task()
				completed = true
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

// failedResult builds the result of a goroutine that did not complete. The recovered
// value is nil only if the function called runtime.Goexit, as t.FailNow does.
func failedResult[T any](recovered any) parallelResult[T] {
	if recovered == nil {
		recovered = "stream: a parallel stage or its source called runtime.Goexit"
	}
	return parallelResult[T]{panicked: true, panicValue: recovered}
}
