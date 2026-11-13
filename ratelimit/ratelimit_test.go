package ratelimit

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

// clock is a manual clock. Sleeping on it advances it, so tests never wait.
type clock struct {
	mu    sync.Mutex
	t     time.Time
	slept []time.Duration
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func (c *clock) sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slept = append(c.slept, d)
	c.t = c.t.Add(d)
	return ctx.Err()
}

func newTest(rate float64, burst int) (*Limiter, *clock) {
	c := &clock{t: time.Unix(1000, 0)}
	l := New(rate, burst)
	l.now, l.sleep, l.last = c.now, c.sleep, c.t
	return l, c
}

func TestNewPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"zero rate":     func() { New(0, 1) },
		"negative rate": func() { New(-1, 1) },
		"NaN rate":      func() { New(math.NaN(), 1) },
		"infinite rate": func() { New(math.Inf(1), 1) },
		"zero burst":    func() { New(1, 0) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New with a %s did not panic", name)
				}
			}()
			f()
		}()
	}
}

func TestAllowBurstAndRefill(t *testing.T) {
	l, c := newTest(10, 3)
	if l.Tokens() != 3 {
		t.Errorf("a new Limiter has %d tokens, want a full bucket", l.Tokens())
	}
	for i := range 3 {
		if !l.Allow() {
			t.Errorf("Allow() %d of the burst was refused", i)
		}
	}
	if l.Allow() || l.Tokens() != 0 {
		t.Errorf("Allow() passed with the bucket empty")
	}
	c.advance(100 * time.Millisecond) // one token at 10 per second
	if !l.Allow() || l.Allow() {
		t.Errorf("after 100ms exactly one more event should pass")
	}
	c.advance(time.Hour)
	if l.Tokens() != 3 {
		t.Errorf("the bucket refilled to %d tokens, want the burst of 3", l.Tokens())
	}
	c.advance(-time.Hour) // the clock steps back
	if l.Tokens() != 3 {
		t.Errorf("a clock stepping back changed the bucket to %d tokens", l.Tokens())
	}
}

func TestWaitQueuesInOrder(t *testing.T) {
	l, c := newTest(10, 1)
	ctx := context.Background()
	if err := l.Wait(ctx); err != nil || len(c.slept) != 0 {
		t.Errorf("the first Wait slept %v, err %v", c.slept, err)
	}
	// The manual clock moves forward by each sleep, so each of these is next in line.
	for i := range 3 {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("Wait %d: %v", i, err)
		}
	}
	for i, d := range c.slept {
		if d < 99*time.Millisecond || d > 101*time.Millisecond {
			t.Errorf("Wait %d slept %v, want about 100ms", i, d)
		}
	}
	if len(c.slept) != 3 {
		t.Errorf("%d sleeps, want 3", len(c.slept))
	}

	// Callers that queue before any of them has slept wait one, two, three intervals.
	l, c = newTest(10, 1)
	l.Allow()
	l.sleep = func(context.Context, time.Duration) error { return nil }
	var waits []time.Duration
	for range 3 {
		before := l.tokens
		l.Wait(ctx)
		waits = append(waits, time.Duration(-(before-1)/l.rate*float64(time.Second)))
	}
	for i, d := range waits {
		if want := time.Duration(i+1) * 100 * time.Millisecond; d != want {
			t.Errorf("queued caller %d waits %v, want %v", i, d, want)
		}
	}
}

func TestWaitContext(t *testing.T) {
	l, _ := newTest(1, 1)
	l.Allow()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Wait(cancelled); !errors.Is(err, context.Canceled) || l.tokens != 0 {
		t.Errorf("Wait with an ended context = %v, tokens %v", err, l.tokens)
	}

	// The deadline comes before a token would: fail at once and take nothing.
	soon, cancel := context.WithDeadline(context.Background(), l.now().Add(10*time.Millisecond))
	defer cancel()
	if err := l.Wait(soon); !errors.Is(err, context.DeadlineExceeded) || l.tokens != 0 {
		t.Errorf("Wait past its deadline = %v, tokens %v", err, l.tokens)
	}

	// The context ends during the sleep: the token goes back.
	l.sleep = func(context.Context, time.Duration) error { return context.Canceled }
	if err := l.Wait(context.Background()); !errors.Is(err, context.Canceled) || l.tokens != 0 {
		t.Errorf("Wait cancelled during its sleep = %v, tokens %v", err, l.tokens)
	}
}

func TestRealClock(t *testing.T) {
	l := New(1000, 1)
	start := time.Now()
	for range 5 {
		if err := l.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 3*time.Millisecond {
		t.Errorf("5 events at 1000 per second took %v, want at least 4ms", elapsed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	slow := New(0.001, 1)
	slow.Allow()
	go cancel()
	if err := slow.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait = %v, want context.Canceled", err)
	}
}

func TestConcurrentUse(t *testing.T) {
	l := New(1e6, 10)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				l.Allow()
				l.Tokens()
				l.Wait(context.Background())
			}
		})
	}
	wg.Wait()
}
