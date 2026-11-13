// Package ratelimit provides Limiter, a token bucket that limits how often something
// happens.
//
//	limiter := ratelimit.New(10, 20) // 10 per second, bursts of up to 20
//	if err := limiter.Wait(ctx); err != nil {
//		return err
//	}
//	callAPI()
//
// The bucket holds up to burst tokens and gains rate tokens per second. Each event
// takes one token, so a Limiter allows short bursts while holding the long-run average
// to the rate. A Limiter is safe for concurrent use.
package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// Limiter is a token bucket. See the package documentation. A Limiter must be created
// with New and must not be copied after first use.
type Limiter struct {
	mu     sync.Mutex
	rate   float64 // tokens per second
	burst  float64
	tokens float64 // may be negative while callers of Wait are queued
	last   time.Time

	// now and sleep are fields so that tests can run without waiting.
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) error
}

// New returns a Limiter that allows rate events per second on average, with bursts of
// up to burst events. It starts full, so the first burst events pass at once. New
// panics unless rate is positive and finite and burst is at least 1.
func New(rate float64, burst int) *Limiter {
	if !(rate > 0) || math.IsInf(rate, 1) {
		panic("ratelimit: New requires a positive, finite rate")
	}
	if burst < 1 {
		panic("ratelimit: New requires a burst of at least 1")
	}
	l := &Limiter{rate: rate, burst: float64(burst), tokens: float64(burst), now: time.Now, sleep: sleepContext}
	l.last = l.now()
	return l
}

// Allow reports whether an event may happen now, and takes a token if so. It never
// waits, so use it to drop or reject what is over the limit.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

// Wait blocks until an event may happen, or until ctx ends, in which case it returns
// the context's error and takes no token. Callers that wait at the same time are served
// in the order they arrived, each a token's worth of time after the one before.
//
// Wait fails at once, without waiting, if ctx has a deadline that comes before its
// turn would.
func (l *Limiter) Wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	l.refill()
	// Taking the token now, even into debt, is what queues later callers behind this
	// one: each sees a larger debt and waits correspondingly longer.
	l.tokens--
	var wait time.Duration
	if l.tokens < 0 {
		wait = time.Duration(-l.tokens / l.rate * float64(time.Second))
	}
	if deadline, ok := ctx.Deadline(); ok && l.now().Add(wait).After(deadline) {
		l.tokens++
		l.mu.Unlock()
		return context.DeadlineExceeded
	}
	l.mu.Unlock()
	if wait <= 0 {
		return nil
	}
	if err := l.sleep(ctx, wait); err != nil {
		l.mu.Lock()
		l.tokens++ // Hand the token back so those queued behind are not held up by it.
		l.mu.Unlock()
		return err
	}
	return nil
}

// Tokens returns how many events could happen right now without waiting.
func (l *Limiter) Tokens() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	return int(max(l.tokens, 0))
}

// refill adds the tokens earned since the last call. The lock must be held.
func (l *Limiter) refill() {
	now := l.now()
	// The clock can step backwards; earn nothing then rather than lose tokens.
	if elapsed := now.Sub(l.last); elapsed > 0 {
		l.tokens = min(l.burst, l.tokens+elapsed.Seconds()*l.rate)
	}
	l.last = now
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
