package stream

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitForGoroutines fails the test if the goroutine count stays above before.
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

// mustPanic returns the value that f panics with.
func mustPanic(t *testing.T, f func()) (value any) {
	t.Helper()
	defer func() {
		value = recover()
		if value == nil {
			t.Error("the function did not panic")
		}
	}()
	f()
	return nil
}

func TestParallelKeepsOrder(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	delays := make([]time.Duration, 200)
	for i := range delays {
		delays[i] = time.Duration(rng.IntN(500)) * time.Microsecond
	}
	got := Range(0, 200).MapConcurrent(16, func(i int) int {
		time.Sleep(delays[i])
		return i * 2
	}).ToSlice()
	want := Range(0, 200).Map(func(i int) int { return i * 2 }).ToSlice()
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestParallelRunsConcurrently(t *testing.T) {
	// Each call waits for the others, so this only finishes if n calls overlap.
	const n = 4
	var arrived sync.WaitGroup
	arrived.Add(n)
	released := make(chan struct{})
	go func() {
		arrived.Wait()
		close(released)
	}()
	got := Range(0, n).MapConcurrent(n, func(i int) int {
		arrived.Done()
		select {
		case <-released:
		case <-time.After(10 * time.Second):
			panic("the calls did not overlap")
		}
		return i
	}).ToSlice()
	if !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Errorf("got %v", got)
	}
}

func TestParallelLimitsConcurrency(t *testing.T) {
	for _, n := range []int{1, 2, 7} {
		var running, peak atomic.Int64
		Range(0, 100).MapConcurrent(n, func(i int) int {
			current := running.Add(1)
			defer running.Add(-1)
			for {
				old := peak.Load()
				if current <= old || peak.CompareAndSwap(old, current) {
					break
				}
			}
			time.Sleep(100 * time.Microsecond)
			return i
		}).Count()
		if got := peak.Load(); got > int64(n) {
			t.Errorf("n = %d: %d calls ran at once", n, got)
		}
	}
}

func TestParallelStages(t *testing.T) {
	var peeked atomic.Int64
	got := Range(0, 10).Parallel(3).
		Filter(IsEven[int]).
		Map(func(i int) int { return i + 1 }).
		Peek(func(int) { peeked.Add(1) }).Map(
		func(i int) string { return fmt.Sprint(i) },
	).ToSlice()
	if want := []string{"1", "3", "5", "7", "9"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := peeked.Load(); got != 5 {
		t.Errorf("Peek ran %d times, want 5", got)
	}
}

func TestParallelFilterSkipsLaterStages(t *testing.T) {
	var mapped atomic.Int64
	got := Range(0, 10).Parallel(2).
		Filter(func(i int) bool { return i < 2 }).
		Map(func(i int) int { mapped.Add(1); return i }).
		ToSlice()
	if !slices.Equal(got, []int{0, 1}) || mapped.Load() != 2 {
		t.Errorf("got %v after %d mapper calls", got, mapped.Load())
	}
}

func TestParallelForEach(t *testing.T) {
	var sum atomic.Int64
	Range(0, 100).Parallel(8).ForEach(func(i int) { sum.Add(int64(i)) })
	if got := sum.Load(); got != 4950 {
		t.Errorf("sum = %d, want 4950", got)
	}
}

func TestParallelSourceRunsOnOneGoroutine(t *testing.T) {
	// The race detector fails this if the supplier runs on more than one goroutine.
	next := 0
	got := Generate(func() int { next++; return next }).Parallel(8).Sequential().Limit(100).ToSlice()
	if want := Range(1, 101).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestParallelInfiniteSourceStops(t *testing.T) {
	before := runtime.NumGoroutine()
	var calls atomic.Int64
	const n = 4
	got := Iterate(0, func(i int) int { return i + 1 }).MapConcurrent(n, func(i int) int {
		calls.Add(1)
		return i
	}).Limit(10).ToSlice()
	if want := Range(0, 10).ToSlice(); !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
	if c := calls.Load(); c > 10+n+2 {
		t.Errorf("%d mapper calls for 10 results", c)
	}
	waitForGoroutines(t, before)
}

func TestParallelBoundsReadAhead(t *testing.T) {
	const n = 3
	var read atomic.Int64
	source := Generate(func() int { return int(read.Add(1)) })
	consumed := int64(0)
	for range source.Parallel(n).Sequential().Limit(20).All() {
		consumed++
		time.Sleep(time.Millisecond) // Give the reader time to get as far ahead as it can.
		if ahead := read.Load() - consumed; ahead > n+2 {
			t.Fatalf("read %d elements past the %d consumed", ahead, consumed)
		}
	}
}

func TestParallelStopsWithoutLeaks(t *testing.T) {
	before := runtime.NumGoroutine()
	s := Range(0, 1000).MapConcurrent(8, func(i int) int { return i })
	if got := s.FindFirst().MustGet(); got != 0 {
		t.Errorf("FindFirst = %d", got)
	}
	for range s.All() {
		break
	}
	waitForGoroutines(t, before)
}

func TestParallelWaitsForRunningCalls(t *testing.T) {
	var running atomic.Int64
	Iterate(0, func(i int) int { return i + 1 }).MapConcurrent(4, func(i int) int {
		running.Add(1)
		defer running.Add(-1)
		time.Sleep(5 * time.Millisecond)
		return i
	}).FindFirst()
	if got := running.Load(); got != 0 {
		t.Errorf("%d calls still running after the terminal operation returned", got)
	}
}

func TestParallelIsReusable(t *testing.T) {
	s := Of(1, 2, 3).MapConcurrent(2, func(i int) int { return i * i })
	for range 3 {
		if got := s.ToSlice(); !slices.Equal(got, []int{1, 4, 9}) {
			t.Errorf("got %v", got)
		}
	}
}

func TestParallelStagePanic(t *testing.T) {
	before := runtime.NumGoroutine()
	var got []int
	value := mustPanic(t, func() {
		Range(0, 100).MapConcurrent(4, func(i int) int {
			if i == 5 {
				panic("boom")
			}
			return i
		}).ForEach(func(i int) { got = append(got, i) })
	})
	if value != "boom" {
		t.Errorf("panic value = %v", value)
	}
	if want := []int{0, 1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("elements before the panic = %v, want %v", got, want)
	}
	waitForGoroutines(t, before)
}

func TestParallelSourcePanic(t *testing.T) {
	before := runtime.NumGoroutine()
	source := FromSeq(func(yield func(int) bool) {
		if yield(1) && yield(2) {
			panic("source failed")
		}
	})
	var got []int
	value := mustPanic(t, func() {
		for v := range source.Parallel(4).Sequential().All() {
			got = append(got, v)
		}
	})
	if value != "source failed" {
		t.Errorf("panic value = %v", value)
	}
	if !slices.Equal(got, []int{1, 2}) {
		t.Errorf("elements before the panic = %v", got)
	}
	waitForGoroutines(t, before)
}

func TestParallelConsumerPanic(t *testing.T) {
	before := runtime.NumGoroutine()
	value := mustPanic(t, func() {
		Range(0, 100).MapConcurrent(4, func(i int) int { return i }).ForEach(func(int) { panic("consumer failed") })
	})
	if value != "consumer failed" {
		t.Errorf("panic value = %v", value)
	}
	waitForGoroutines(t, before)
}

func TestParallelGoexit(t *testing.T) {
	before := runtime.NumGoroutine()
	stage := mustPanic(t, func() {
		Of(1).MapConcurrent(2, func(i int) int {
			runtime.Goexit()
			return i
		}).Count()
	})
	source := mustPanic(t, func() {
		FromSeq(func(func(int) bool) { runtime.Goexit() }).Parallel(2).ToSlice()
	})
	for name, value := range map[string]any{"stage": stage, "source": source} {
		if value == nil {
			t.Errorf("%s: Goexit was not reported", name)
		}
	}
	waitForGoroutines(t, before)
}

func TestParallelEmptyAndZeroValue(t *testing.T) {
	if got := Empty[int]().Parallel(2).ToSlice(); got == nil || len(got) != 0 {
		t.Errorf("empty source = %#v", got)
	}
	var zero ParallelStream[int]
	if got := zero.Filter(IsEven[int]).Map(func(i int) int { return i }).Sequential().Count(); got != 0 {
		t.Errorf("zero value gave %d elements", got)
	}
	if got := zero.Map(func(i int) string { return "" }).ToSlice(); len(got) != 0 {
		t.Errorf("ParallelMap of the zero value = %v", got)
	}
}

func TestParallelSizeHint(t *testing.T) {
	s := Of(1, 2, 3).Parallel(2)
	if got := s.Map(func(i int) int { return i }).Sequential().size; got != 3 {
		t.Errorf("size after Map = %d, want 3", got)
	}
	if got := s.Filter(IsEven[int]).Sequential().size; got != -1 {
		t.Errorf("size after Filter = %d, want -1", got)
	}
}

func TestParallelRejectsBadArguments(t *testing.T) {
	inc := func(i int) int { return i + 1 }
	p := Of(1).Parallel(1)
	for name, f := range map[string]func(){
		"zero n":             func() { Of(1).Parallel(0) },
		"negative n":         func() { Of(1).Parallel(-1) },
		"MapConcurrent n":    func() { Of(1).MapConcurrent(0, inc) },
		"MapConcurrent func": func() { Of(1).MapConcurrent[int](1, nil) },
		"Filter":             func() { p.Filter(nil) },
		"Map":                func() { p.Map[int](nil) },
		"Peek":               func() { p.Peek(nil) },
		"ForEach":            func() { p.ForEach(nil) },
	} {
		t.Run(name, func(t *testing.T) { mustPanic(t, f) })
	}
}

// spin stands in for CPU-bound work of about the given number of iterations.
func spin(i, iterations int) int {
	for range iterations {
		i = i*31 + 7
	}
	return i
}

func benchmarkMap(b *testing.B, iterations int) {
	values := Range(0, 1000)
	mapper := func(i int) int { return spin(i, iterations) }
	b.Run("sequential", func(b *testing.B) {
		for b.Loop() {
			values.Map(mapper).Count()
		}
	})
	b.Run("parallel", func(b *testing.B) {
		n := runtime.GOMAXPROCS(0)
		for b.Loop() {
			values.MapConcurrent(n, mapper).Count()
		}
	})
}

func BenchmarkMap1Iteration(b *testing.B)     { benchmarkMap(b, 1) }
func BenchmarkMap1kIterations(b *testing.B)   { benchmarkMap(b, 1_000) }
func BenchmarkMap3kIterations(b *testing.B)   { benchmarkMap(b, 3_000) }
func BenchmarkMap10kIterations(b *testing.B)  { benchmarkMap(b, 10_000) }
func BenchmarkMap100kIterations(b *testing.B) { benchmarkMap(b, 100_000) }
