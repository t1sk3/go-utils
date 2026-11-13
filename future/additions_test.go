package future

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", name)
		}
	}()
	f()
}

func TestGoCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	fut := GoCtx(ctx, func(ctx context.Context) (int, error) {
		close(started)
		<-ctx.Done()
		return 0, ctx.Err()
	})
	<-started
	if fut.IsDone() {
		t.Errorf("the Future completed before its function returned")
	}
	cancel()
	if _, err := fut.Get(context.Background()); !errors.Is(err, context.Canceled) {
		t.Errorf("Get after cancelling = %v", err)
	}
	ran := false
	late := GoCtx(ctx, func(context.Context) (int, error) { ran = true; return 1, nil })
	if _, err := late.Get(context.Background()); !errors.Is(err, context.Canceled) || ran {
		t.Errorf("GoCtx with an ended context = %v, ran = %v", err, ran)
	}
	if v, err := GoCtx(context.Background(), func(context.Context) (int, error) { return 7, nil }).Get(context.Background()); v != 7 || err != nil {
		t.Errorf("GoCtx = %v, %v", v, err)
	}
	mustPanic(t, "GoCtx(nil)", func() { GoCtx[int](context.Background(), nil) })
}

func TestIsDoneAndTryGet(t *testing.T) {
	release := make(chan struct{})
	fut := Go(func() (int, error) { <-release; return 5, nil })
	if _, ok := fut.TryGet(); ok || fut.IsDone() {
		t.Errorf("a running Future reports a result")
	}
	close(release)
	<-fut.Done()
	if r, ok := fut.TryGet(); !ok || r.OrElse(0) != 5 || !fut.IsDone() {
		t.Errorf("TryGet of a finished Future = %v, %v", r, ok)
	}
	if r, ok := Failed[int](errBoom).TryGet(); !ok || r.Err() != errBoom {
		t.Errorf("TryGet of a failed Future = %v, %v", r, ok)
	}
}

func TestRecoverAndOrElse(t *testing.T) {
	ctx := context.Background()
	called := false
	ok := Completed(1).Recover(func(error) (int, error) { called = true; return 9, nil })
	if v, err := ok.Get(ctx); v != 1 || err != nil || called {
		t.Errorf("Recover of a success = %v, %v, called = %v", v, err, called)
	}
	if v, err := Failed[int](errBoom).Recover(func(err error) (int, error) { return len(err.Error()), nil }).Get(ctx); v != 4 || err != nil {
		t.Errorf("Recover of a failure = %v, %v", v, err)
	}
	other := errors.New("other")
	if _, err := Failed[int](errBoom).Recover(func(error) (int, error) { return 0, other }).Get(ctx); err != other {
		t.Errorf("Recover that fails itself = %v", err)
	}
	if v, _ := Failed[int](errBoom).OrElse(3).Get(ctx); v != 3 {
		t.Errorf("OrElse of a failure = %v", v)
	}
	if v, _ := Completed(1).OrElse(3).Get(ctx); v != 1 {
		t.Errorf("OrElse of a success = %v", v)
	}
	mustPanic(t, "Recover(nil)", func() { Completed(1).Recover(nil) })
}

func TestAllSettled(t *testing.T) {
	ctx := context.Background()
	release := make(chan struct{})
	slow := Go(func() (int, error) { <-release; return 3, nil })
	go close(release)
	results, err := AllSettled(ctx, Completed(1), Failed[int](errBoom), slow)
	if err != nil || len(results) != 3 {
		t.Fatalf("AllSettled = %v, %v", results, err)
	}
	if results[0].OrElse(0) != 1 || results[1].Err() != errBoom || results[2].OrElse(0) != 3 {
		t.Errorf("results = %v", results)
	}
	if results, err := AllSettled[int](ctx); err != nil || len(results) != 0 {
		t.Errorf("AllSettled of nothing = %v, %v", results, err)
	}

	ended, cancel := context.WithCancel(ctx)
	cancel()
	never := Go(func() (int, error) { select {} })
	if results, err := AllSettled(ended, Completed(1), never); !errors.Is(err, context.Canceled) || results != nil {
		t.Errorf("AllSettled with a pending future and an ended context = %v, %v", results, err)
	}
	// Outcomes that are already there are returned even if the context has ended.
	for range 50 {
		if results, err := AllSettled(ended, Completed(1), Failed[int](errBoom)); err != nil || len(results) != 2 {
			t.Fatalf("AllSettled of finished futures with an ended context = %v, %v", results, err)
		}
	}
	mustPanic(t, "AllSettled with a nil Future", func() { AllSettled(ctx, Completed(1), nil) })
}

func TestPromise(t *testing.T) {
	ctx := context.Background()
	p := NewPromise[string]()
	first, second := p.Future(), p.Future()
	if first != second || first.IsDone() {
		t.Errorf("a new Promise is done, or hands out different Futures")
	}
	waiter := p.Future().Then(func(s string) (int, error) { return len(s), nil })
	if !p.Complete("hello") || p.Complete("again") || p.Fail(errBoom) {
		t.Errorf("only the first completion should report true")
	}
	if v, err := p.Future().Get(ctx); v != "hello" || err != nil {
		t.Errorf("Get = %q, %v", v, err)
	}
	if n, err := waiter.Get(ctx); n != 5 || err != nil {
		t.Errorf("a Future chained before completion = %v, %v", n, err)
	}

	failed := NewPromise[int]()
	if !failed.Fail(errBoom) || failed.Complete(1) {
		t.Errorf("Fail then Complete reported the wrong winner")
	}
	if v, err := failed.Future().Get(ctx); v != 0 || err != errBoom {
		t.Errorf("Get of a failed Promise = %v, %v", v, err)
	}
	mustPanic(t, "Fail(nil)", func() { NewPromise[int]().Fail(nil) })

	// Many goroutines race to complete: exactly one wins and its value is the result.
	race := NewPromise[int]()
	var wins atomic.Int32
	var winner atomic.Int32
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Go(func() {
			if race.Complete(i) {
				wins.Add(1)
				winner.Store(int32(i))
			}
		})
	}
	wg.Wait()
	if v, _ := race.Future().Get(ctx); wins.Load() != 1 || v != int(winner.Load()) {
		t.Errorf("%d goroutines completed the Promise; it holds %d, the winner set %d", wins.Load(), v, winner.Load())
	}
}

func TestPool(t *testing.T) {
	const limit = 3
	pool := NewPool(limit)
	var running, peak atomic.Int32
	futures := make([]*Future[int], 20)
	for i := range futures {
		futures[i] = pool.Go(func() (int, error) {
			n := running.Add(1)
			for {
				if old := peak.Load(); n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			running.Add(-1)
			return i * 2, nil
		})
	}
	values, err := All(context.Background(), futures...)
	if err != nil || len(values) != 20 || values[7] != 14 {
		t.Fatalf("All = %v, %v", values, err)
	}
	if got := peak.Load(); got > limit || got < 1 {
		t.Errorf("%d functions ran at once, want at most %d", got, limit)
	}

	// A panic frees its slot, and Wait returns once everything has.
	bad := pool.Go(func() (int, error) { panic("boom") })
	text := pool.Go(func() (string, error) { return "other type", nil })
	pool.Wait()
	if !bad.IsDone() || !text.IsDone() {
		t.Errorf("Wait returned before every function had")
	}
	if _, err := bad.Get(context.Background()); err == nil {
		t.Errorf("a panic in a pooled function was lost")
	}
	if v, _ := pool.Go(func() (int, error) { return 1, nil }).Get(context.Background()); v != 1 {
		t.Errorf("the Pool is stuck after a panic")
	}
	mustPanic(t, "NewPool(0)", func() { NewPool(0) })
	mustPanic(t, "Pool.Go(nil)", func() { pool.Go[int](nil) })
}
