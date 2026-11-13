package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

// record replaces sleep with a function that returns at once and notes each delay.
func record(t *testing.T) *[]time.Duration {
	t.Helper()
	var delays []time.Duration
	realSleep, realRandom := sleep, random
	sleep = func(ctx context.Context, d time.Duration) error {
		delays = append(delays, d)
		return ctx.Err()
	}
	t.Cleanup(func() { sleep, random = realSleep, realRandom })
	return &delays
}

// failing returns a function that fails the first n times it is called.
func failing(n int, calls *int) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		*calls++
		if *calls <= n {
			return "partial", errBoom
		}
		return "done", nil
	}
}

func TestSucceedsAfterRetries(t *testing.T) {
	delays := record(t)
	calls := 0
	got, err := Do(context.Background(), Policy{Attempts: 5, Delay: time.Second}, failing(2, &calls))
	if got != "done" || err != nil || calls != 3 {
		t.Errorf("Do = %q, %v after %d calls", got, err, calls)
	}
	if !slices.Equal(*delays, []time.Duration{time.Second, time.Second}) {
		t.Errorf("delays = %v, want a constant second", *delays)
	}
}

func TestGivesUp(t *testing.T) {
	record(t)
	calls := 0
	got, err := Do(context.Background(), Policy{}, failing(10, &calls))
	if got != "" || err != errBoom || calls != 3 {
		t.Errorf("Do with the zero Policy = %q, %v after %d calls, want 3 attempts and no value", got, err, calls)
	}
	calls = 0
	if _, err := Do(context.Background(), Policy{Attempts: 1}, failing(10, &calls)); err != errBoom || calls != 1 {
		t.Errorf("Attempts: 1 made %d calls", calls)
	}
}

func TestBackoff(t *testing.T) {
	delays := record(t)
	calls := 0
	Do(context.Background(), Policy{Attempts: 6, Delay: time.Second, Multiplier: 2, MaxDelay: 5 * time.Second}, failing(10, &calls))
	want := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second}
	if !slices.Equal(*delays, want) {
		t.Errorf("delays = %v, want %v", *delays, want)
	}

	// A delay that outgrows a Duration stays at the largest one instead of wrapping.
	if got := next(1<<62, Policy{Multiplier: 4}); got != 1<<63-1 {
		t.Errorf("next past the largest Duration = %v", got)
	}
	if got := next(time.Second, Policy{Multiplier: 0.5}); got != time.Second {
		t.Errorf("a Multiplier below 1 changed the delay to %v", got)
	}
}

func TestJitter(t *testing.T) {
	record(t)
	for r, want := range map[float64]time.Duration{0: 800 * time.Millisecond, 0.5: time.Second, 1: 1200 * time.Millisecond} {
		random = func() float64 { return r }
		if got := jitter(time.Second, 0.2); got != want {
			t.Errorf("jitter with random %v = %v, want %v", r, got, want)
		}
	}
	random = func() float64 { return 0 }
	if got := jitter(time.Second, 5); got != 0 {
		t.Errorf("a Jitter above 1 was not clamped: %v", got)
	}
	// Spreading the longest delay upwards must not wrap around to a negative one.
	random = func() float64 { return 1 }
	if got := jitter(math.MaxInt64, 1); got != math.MaxInt64 {
		t.Errorf("jitter of the longest delay = %v", got)
	}
	if jitter(time.Second, 0) != time.Second || jitter(time.Second, -1) != time.Second || jitter(0, 0.5) != 0 {
		t.Errorf("jitter changed a delay it should leave alone")
	}
}

func TestPermanentAndRetryIf(t *testing.T) {
	record(t)
	calls := 0
	_, err := Do(context.Background(), Policy{Attempts: 5}, func(context.Context) (int, error) {
		calls++
		return 0, Permanent(errBoom)
	})
	if err != errBoom || calls != 1 {
		t.Errorf("a Permanent error gave %v after %d calls, want errBoom itself after 1", err, calls)
	}
	// A Permanent error that the function wrapped further stops the retries too, and
	// comes back with that wrapping.
	calls = 0
	_, err = Do(context.Background(), Policy{Attempts: 5}, func(context.Context) (int, error) {
		calls++
		return 0, fmt.Errorf("loading: %w", Permanent(errBoom))
	})
	if calls != 1 || err == nil || err.Error() != "loading: boom" || !errors.Is(err, errBoom) {
		t.Errorf("a wrapped Permanent error gave %v after %d calls", err, calls)
	}
	if Permanent(nil) != nil {
		t.Errorf("Permanent(nil) is not nil")
	}
	if p := Permanent(errBoom); p.Error() != "boom" || !errors.Is(p, errBoom) {
		t.Errorf("Permanent(errBoom) = %v", p)
	}

	calls = 0
	policy := Policy{Attempts: 5, RetryIf: func(err error) bool { return calls < 2 }}
	if _, err := Do(context.Background(), policy, failing(10, &calls)); err != errBoom || calls != 2 {
		t.Errorf("RetryIf gave %v after %d calls, want 2", err, calls)
	}
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := Do(ctx, Policy{Attempts: 5, Delay: time.Hour}, func(context.Context) (int, error) {
		calls++
		cancel()
		return 0, errBoom
	})
	if err != errBoom || calls != 1 {
		t.Errorf("a context that ended during the call gave %v after %d calls", err, calls)
	}

	// The context ends during the wait: the real sleep must return, and both errors show.
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	calls = 0
	start := time.Now()
	_, err = Do(ctx, Policy{Attempts: 5, Delay: time.Hour}, failing(10, &calls))
	if !errors.Is(err, errBoom) || !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Errorf("a context that ended during the wait gave %v after %d calls", err, calls)
	}
	if time.Since(start) > 30*time.Second {
		t.Errorf("Do did not stop waiting when the context ended")
	}

	// The real sleep waits a short delay out, and returns at once for none.
	calls = 0
	if got, err := Do(context.Background(), Policy{Delay: time.Millisecond}, failing(1, &calls)); got != "done" || err != nil {
		t.Errorf("Do with a real delay = %q, %v", got, err)
	}
	if err := sleep(context.Background(), 0); err != nil {
		t.Errorf("sleep(0) = %v", err)
	}
}

func TestNilFunctionPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("Do(nil) did not panic")
		}
	}()
	Do[int](context.Background(), Policy{}, nil)
}
