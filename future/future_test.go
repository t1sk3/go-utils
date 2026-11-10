package future

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

// blocked returns a Future that completes with value when the returned function is
// called.
func blocked[T any](value T, err error) (*Future[T], func()) {
	release := make(chan struct{})
	return Go(func() (T, error) {
		<-release
		return value, err
	}), sync.OnceFunc(func() { close(release) })
}

func waitForGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines running, started with %d", runtime.NumGoroutine(), before)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestGoReturnsResult(t *testing.T) {
	ctx := context.Background()
	fut := Go(func() (int, error) { return 1, nil })
	for range 2 { // The result can be read repeatedly.
		if v, err := fut.Get(ctx); v != 1 || err != nil {
			t.Errorf("Get = %d, %v", v, err)
		}
	}
	failed := Go(func() (int, error) { return 0, errBoom })
	if _, err := failed.Get(ctx); !errors.Is(err, errBoom) {
		t.Errorf("error = %v", err)
	}
}

func TestGoKeepsValueAlongsideError(t *testing.T) {
	fut := Go(func() (int, error) { return 7, errBoom })
	if v, err := fut.Get(context.Background()); v != 7 || !errors.Is(err, errBoom) {
		t.Errorf("Get = %d, %v", v, err)
	}
}

func TestGoRecoversPanic(t *testing.T) {
	fut := Go(func() (int, error) { panic(errBoom) })
	_, err := fut.Get(context.Background())
	pe, ok := errors.AsType[*PanicError](err)
	if !ok {
		t.Fatalf("error = %v, want a *PanicError", err)
	}
	if pe.Value != errBoom || !errors.Is(err, errBoom) {
		t.Errorf("Value = %v, errors.Is = %v", pe.Value, errors.Is(err, errBoom))
	}
	if !strings.Contains(err.Error(), "boom") || !strings.Contains(string(pe.Stack), "future_test") {
		t.Errorf("message or stack is missing details:\n%v", err)
	}
	plain := Go(func() (int, error) { panic("text") })
	_, err = plain.Get(context.Background())
	if pe, ok := errors.AsType[*PanicError](err); !ok || pe.Value != "text" || pe.Unwrap() != nil {
		t.Errorf("non-error panic: %#v", err)
	}
}

func TestGoReportsGoexit(t *testing.T) {
	fut := Go(func() (int, error) {
		runtime.Goexit()
		return 1, nil
	})
	_, err := fut.Get(context.Background())
	if _, ok := errors.AsType[*PanicError](err); !ok {
		t.Errorf("error = %v, want a *PanicError", err)
	}
}

func TestGetContext(t *testing.T) {
	fut, release := blocked(1, nil)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if v, err := fut.Get(ctx); v != 0 || !errors.Is(err, context.Canceled) {
		t.Errorf("Get = %d, %v", v, err)
	}

	// Cancellation during the wait.
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()
	if _, err := fut.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v", err)
	}

	// An available result wins over an ended context.
	if v, err := Completed(5).Get(ctx); v != 5 || err != nil {
		t.Errorf("Get on a completed Future = %d, %v", v, err)
	}
}

func TestDone(t *testing.T) {
	fut, release := blocked(1, nil)
	select {
	case <-fut.Done():
		t.Error("Done closed before the result was ready")
	default:
	}
	release()
	<-fut.Done()
	if v, err := fut.Get(context.Background()); v != 1 || err != nil {
		t.Errorf("Get = %d, %v", v, err)
	}
}

func TestCompletedAndFailed(t *testing.T) {
	ctx := context.Background()
	if v, err := Completed("a").Get(ctx); v != "a" || err != nil {
		t.Errorf("Completed = %q, %v", v, err)
	}
	if _, err := Failed[string](errBoom).Get(ctx); !errors.Is(err, errBoom) {
		t.Errorf("Failed = %v", err)
	}
}

func TestThen(t *testing.T) {
	ctx := context.Background()
	// The method can change the type at each step, so calls chain.
	label, err := Completed("abc").
		Then(func(s string) (int, error) { return len(s), nil }).
		Then(func(n int) (string, error) { return fmt.Sprintf("len=%d", n), nil }).
		Get(ctx)
	if label != "len=3" || err != nil {
		t.Errorf("Then chain = %q, %v", label, err)
	}
	var calls atomic.Int64
	_, err = Failed[string](errBoom).
		Then(func(s string) (int, error) { calls.Add(1); return 0, nil }).
		Then(func(n int) (int, error) { calls.Add(1); return n, nil }).
		Get(ctx)
	if !errors.Is(err, errBoom) || calls.Load() != 0 {
		t.Errorf("error = %v after %d calls", err, calls.Load())
	}
	if _, err := Completed(1).Then(func(int) (int, error) { panic(errBoom) }).Get(ctx); !errors.Is(err, errBoom) {
		t.Errorf("panic in Then = %v", err)
	}
}

func TestAll(t *testing.T) {
	ctx := context.Background()
	slow, release := blocked(1, nil)
	go release()
	got, err := All(ctx, slow, Completed(2), Go(func() (int, error) { return 3, nil }))
	if err != nil || !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("All = %v, %v", got, err)
	}
	if got, err := All[int](ctx); err != nil || got == nil || len(got) != 0 {
		t.Errorf("All of nothing = %#v, %v", got, err)
	}
}

func TestAllReturnsOnFirstFailure(t *testing.T) {
	before := runtime.NumGoroutine()
	never, release := blocked(1, nil)
	defer release()
	got, err := All(context.Background(), never, Failed[int](errBoom))
	if got != nil || !errors.Is(err, errBoom) {
		t.Errorf("All = %v, %v", got, err)
	}
	release()
	<-never.Done()
	waitForGoroutines(t, before)
}

func TestAllSeesLastFutureFailing(t *testing.T) {
	for range 1000 {
		if _, err := All(context.Background(), Failed[int](errBoom)); !errors.Is(err, errBoom) {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestAllContext(t *testing.T) {
	before := runtime.NumGoroutine()
	never, release := blocked(1, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := All(ctx, never); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v", err)
	}
	release()
	<-never.Done()
	waitForGoroutines(t, before)
}

func TestAny(t *testing.T) {
	ctx := context.Background()
	never, release := blocked(1, nil)
	defer release()
	if v, err := Any(ctx, never, Failed[int](errBoom), Completed(2)); v != 2 || err != nil {
		t.Errorf("Any = %d, %v", v, err)
	}
}

func TestAnyAllFail(t *testing.T) {
	errOther := errors.New("other")
	v, err := Any(context.Background(), Failed[int](errBoom), Failed[int](errOther))
	if v != 0 || !errors.Is(err, errBoom) || !errors.Is(err, errOther) {
		t.Errorf("Any = %d, %v", v, err)
	}
	if want := "boom\nother"; err.Error() != want {
		t.Errorf("error = %q, want errors in the order given: %q", err, want)
	}
}

func TestAnyNoFuturesAndContext(t *testing.T) {
	if _, err := Any[int](context.Background()); !errors.Is(err, ErrNoFutures) {
		t.Errorf("error = %v", err)
	}
	never, release := blocked(1, nil)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Any(ctx, never); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v", err)
	}
}

func TestNilFunctions(t *testing.T) {
	for name, f := range map[string]func(){
		"Go":   func() { Go[int](nil) },
		"Then": func() { Completed(1).Then[int](nil) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			f()
		}()
	}
}
