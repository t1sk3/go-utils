// Package retry runs a function again when it fails, with a delay between attempts.
//
//	page, err := retry.Do(ctx, retry.Policy{Attempts: 5, Delay: 100 * time.Millisecond, Multiplier: 2},
//		func(ctx context.Context) (Page, error) { return fetch(ctx, url) })
//
// Retrying only helps for failures that may pass, such as a timeout or a busy server.
// Wrap an error in Permanent, or set Policy.RetryIf, to give up at once on the rest.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"time"
)

// Policy says how often and how patiently to retry. The zero value makes three
// attempts with no delay between them.
type Policy struct {
	// Attempts is the most times the function is called, including the first. Zero or
	// less means 3.
	Attempts int
	// Delay is the wait before the second attempt. Zero means no wait.
	Delay time.Duration
	// Multiplier scales the delay after each attempt: 2 doubles it, for exponential
	// backoff. Less than 1, including zero, keeps the delay constant.
	Multiplier float64
	// MaxDelay limits how far Multiplier can grow the delay. Zero means no limit.
	MaxDelay time.Duration
	// Jitter spreads each delay randomly by up to this fraction in either direction,
	// so that many clients that failed together do not retry together: 0.2 turns a
	// delay of 1s into something between 0.8s and 1.2s. It is clamped to [0, 1].
	Jitter float64
	// RetryIf reports whether an error is worth another attempt. Nil means every error
	// is, except those wrapped in Permanent and the context's own.
	RetryIf func(error) bool
}

// Do calls f until it succeeds, the Policy gives up, or ctx ends, and returns the
// result of the last call. f receives ctx and should pass it on, so that an attempt in
// progress stops when ctx ends.
//
// When ctx ends while waiting, Do returns the last error from f joined with the
// context's, so errors.Is finds both. Do panics if f is nil.
func Do[T any](ctx context.Context, policy Policy, f func(context.Context) (T, error)) (T, error) {
	if f == nil {
		panic("retry: Do requires a non-nil function")
	}
	attempts := policy.Attempts
	if attempts <= 0 {
		attempts = 3
	}
	delay := policy.Delay
	for attempt := 1; ; attempt++ {
		v, err := f(ctx)
		if err == nil {
			return v, nil
		}
		var zero T
		if p, ok := err.(*permanentError); ok {
			return zero, p.err
		}
		if _, ok := errors.AsType[*permanentError](err); ok {
			// f wrapped a Permanent error in context of its own; keep that context.
			return zero, err
		}
		if attempt == attempts || ctx.Err() != nil || (policy.RetryIf != nil && !policy.RetryIf(err)) {
			return zero, err
		}
		if waitErr := sleep(ctx, jitter(delay, policy.Jitter)); waitErr != nil {
			return zero, errors.Join(err, waitErr)
		}
		delay = next(delay, policy)
	}
}

// Permanent wraps err so that Do returns it at once instead of retrying. Do returns
// err itself, without the wrapper; if the function wrapped the Permanent error further,
// as with fmt.Errorf and %w, Do returns that error as it is. Permanent returns nil if
// err is nil.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// next returns the delay to use after one of the given length.
func next(delay time.Duration, policy Policy) time.Duration {
	if policy.Multiplier > 1 {
		delay = toDuration(float64(delay) * policy.Multiplier)
	}
	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}
	return delay
}

// toDuration converts nanoseconds to a Duration. A value that does not fit becomes the
// longest Duration there is, because converting it directly would give an unspecified
// result, possibly a negative one that turns a long wait into none at all.
func toDuration(nanos float64) time.Duration {
	if nanos >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	return time.Duration(nanos)
}

// jitter spreads delay by up to the given fraction in either direction.
func jitter(delay time.Duration, fraction float64) time.Duration {
	fraction = min(max(fraction, 0), 1)
	if fraction == 0 || delay <= 0 {
		return delay
	}
	spread := (random()*2 - 1) * fraction
	return toDuration(float64(delay) * (1 + spread))
}

// random and sleep are variables so that tests can run without waiting.
var (
	random = rand.Float64
	sleep  = func(ctx context.Context, d time.Duration) error {
		if d <= 0 {
			return ctx.Err()
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
)
