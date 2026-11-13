package stream

import (
	"cmp"
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/t1sk3/go-utils/types/pair"
	"github.com/t1sk3/go-utils/types/result"
)

// lazy returns s as a stream that does not come straight from a slice, so that the
// general paths run instead of the slice shortcuts.
func lazy[T any](s Stream[T]) Stream[T] { return FromSeq(s.All()) }

func TestReverseIntersperseCycle(t *testing.T) {
	if got := Of(1, 2, 3).Reverse().ToSlice(); !slices.Equal(got, []int{3, 2, 1}) {
		t.Errorf("Reverse = %v", got)
	}
	if got := Range(0, 5).Reverse().Limit(2).ToSlice(); !slices.Equal(got, []int{4, 3}) {
		t.Errorf("Reverse then Limit = %v", got)
	}
	if Empty[int]().Reverse().Count() != 0 {
		t.Errorf("Reverse of an empty stream has elements")
	}

	if got := Of("a", "b", "c").Intersperse("-").ToSlice(); !slices.Equal(got, []string{"a", "-", "b", "-", "c"}) {
		t.Errorf("Intersperse = %v", got)
	}
	if got := Of("a").Intersperse("-").ToSlice(); !slices.Equal(got, []string{"a"}) {
		t.Errorf("Intersperse of one element = %v", got)
	}
	if Empty[string]().Intersperse("-").Count() != 0 {
		t.Errorf("Intersperse of an empty stream has elements")
	}
	// Stopping on an element and on a separator.
	for n, want := range map[int][]int{1: {1}, 2: {1, 0}, 3: {1, 0, 2}} {
		if got := Range(1, 9).Intersperse(0).Limit(n).ToSlice(); !slices.Equal(got, want) {
			t.Errorf("Intersperse then Limit(%d) = %v", n, got)
		}
	}

	if got := Of(1, 2, 3).Cycle().Limit(8).ToSlice(); !slices.Equal(got, []int{1, 2, 3, 1, 2, 3, 1, 2}) {
		t.Errorf("Cycle = %v", got)
	}
	if got := Of(1, 2, 3).Cycle().Limit(2).ToSlice(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Cycle stopped during the first pass = %v", got)
	}
	if Empty[int]().Cycle().Count() != 0 {
		t.Errorf("Cycle of an empty stream did not end")
	}
	// The source is read once even though its elements repeat.
	reads := 0
	once := Range(0, 2).Peek(func(int) { reads++ })
	if got := once.Cycle().Limit(7).Count(); got != 7 || reads != 2 {
		t.Errorf("Cycle gave %d elements from %d reads of the source", got, reads)
	}
}

func TestTakeLast(t *testing.T) {
	for _, src := range []Stream[int]{Range(0, 10), Of(0, 1, 2, 3, 4, 5, 6, 7, 8, 9), lazy(Range(0, 10))} {
		for n, want := range map[int][]int{0: {}, 1: {9}, 3: {7, 8, 9}, 10: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, 99: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9}} {
			if got := src.TakeLast(n).ToSlice(); !slices.Equal(got, want) {
				t.Errorf("TakeLast(%d) = %v, want %v", n, got, want)
			}
		}
		if got := src.TakeLast(4).Limit(2).ToSlice(); !slices.Equal(got, []int{6, 7}) {
			t.Errorf("TakeLast then Limit = %v", got)
		}
	}
	if Empty[int]().TakeLast(3).Count() != 0 || lazy(Empty[int]()).TakeLast(3).Count() != 0 {
		t.Errorf("TakeLast of an empty stream has elements")
	}
	mustPanic(t, func() { Of(1).TakeLast(-1) })
	// A huge n must not turn into a huge buffer.
	if got := lazy(Of(1, 2)).TakeLast(math.MaxInt).ToSlice(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("TakeLast with a huge n = %v", got)
	}
}

func TestTop(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	values := make([]int, 500)
	for i := range values {
		values[i] = rng.IntN(10_000)
	}
	for _, n := range []int{0, 1, 7, 500, 900} {
		want := FromSlice(values).Sorted(cmp.Compare[int]).Limit(n).ToSlice()
		if got := FromSlice(values).Top(n, cmp.Compare[int]).ToSlice(); !slices.Equal(got, want) {
			t.Errorf("Top(%d) differs from Sorted.Limit", n)
		}
	}
	largest := FromSlice(values).Top(3, Reversed(cmp.Compare[int])).ToSlice()
	if want := FromSlice(values).Sorted(Reversed(cmp.Compare[int])).Limit(3).ToSlice(); !slices.Equal(largest, want) {
		t.Errorf("Top with a reversed comparator = %v, want %v", largest, want)
	}
	if got := Range(0, 100).Top(5, cmp.Compare[int]).Limit(2).ToSlice(); !slices.Equal(got, []int{0, 1}) {
		t.Errorf("Top then Limit = %v", got)
	}
	if Empty[int]().Top(3, cmp.Compare[int]).Count() != 0 {
		t.Errorf("Top of an empty stream has elements")
	}
	mustPanic(t, func() { Of(1).Top(-1, cmp.Compare[int]) })
	mustPanic(t, func() { Of(1).Top(1, nil) })
}

func TestMinByMaxBy(t *testing.T) {
	type user struct {
		name string
		age  int
	}
	users := Of(user{"ann", 30}, user{"bob", 25}, user{"cy", 41}, user{"di", 25}, user{"ed", 41})
	age := func(u user) int { return u.age }
	if got := users.MinBy(age).MustGet().name; got != "bob" {
		t.Errorf("MinBy = %s, want the first of the youngest", got)
	}
	if got := users.MaxBy(age).MustGet().name; got != "cy" {
		t.Errorf("MaxBy = %s, want the first of the oldest", got)
	}
	if Empty[user]().MinBy(age).IsPresent() || Empty[user]().MaxBy(age).IsPresent() {
		t.Errorf("MinBy or MaxBy of an empty stream returned an element")
	}
	calls := 0
	users.MaxBy(func(u user) int { calls++; return u.age })
	if calls != 5 {
		t.Errorf("the key function ran %d times for 5 elements", calls)
	}
	mustPanic(t, func() { users.MinBy[int](nil) })
	mustPanic(t, func() { users.MaxBy[int](nil) })
}

func TestZipWithUnzipFlatten(t *testing.T) {
	label := func(n int, s string) string { return s + strconv.Itoa(n) }
	for name, b := range map[string]Stream[string]{"slice": Of("a", "b"), "iterator": lazy(Of("a", "b"))} {
		if got := Range(1, 100).ZipWith(b, label).ToSlice(); !slices.Equal(got, []string{"a1", "b2"}) {
			t.Errorf("%s: ZipWith = %v", name, got)
		}
		if got := Of(1).ZipWith(b, label).ToSlice(); !slices.Equal(got, []string{"a1"}) {
			t.Errorf("%s: ZipWith with a shorter receiver = %v", name, got)
		}
		if got := Range(1, 100).ZipWith(b, label).Limit(1).ToSlice(); !slices.Equal(got, []string{"a1"}) {
			t.Errorf("%s: ZipWith then Limit = %v", name, got)
		}
	}
	if Range(1, 100).ZipWith(Empty[string](), label).Count() != 0 || Range(1, 100).ZipWith(lazy(Empty[string]()), label).Count() != 0 {
		t.Errorf("ZipWith an empty stream has elements")
	}
	mustPanic(t, func() { Of(1).ZipWith[string, string](Of("a"), nil) })

	as, bs := Unzip(Of(pair.Of(1, "a"), pair.Of(2, "b")))
	if !slices.Equal(as, []int{1, 2}) || !slices.Equal(bs, []string{"a", "b"}) {
		t.Errorf("Unzip = %v, %v", as, bs)
	}
	if as, bs := Unzip(Empty[pair.Pair[int, string]]()); as == nil || bs == nil || len(as)+len(bs) != 0 {
		t.Errorf("Unzip of an empty stream = %#v, %#v", as, bs)
	}
	if as, bs := Unzip(Range(0, 3).Zip(Of("x", "y", "z"))); !slices.Equal(as, []int{0, 1, 2}) || !slices.Equal(bs, []string{"x", "y", "z"}) {
		t.Errorf("Unzip of Zip = %v, %v", as, bs)
	}

	nested := Of(Of(1, 2), Empty[int](), Range(3, 5))
	if got := Flatten(nested).ToSlice(); !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Errorf("Flatten = %v", got)
	}
	if got := Flatten(nested).Limit(3).ToSlice(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("Flatten then Limit = %v", got)
	}
}

func TestTryMapAndCollectResults(t *testing.T) {
	got, err := CollectResults(Of("1", "2", "3").TryMap(strconv.Atoi))
	if err != nil || !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("CollectResults = %v, %v", got, err)
	}
	if got, err := CollectResults(Empty[string]().TryMap(strconv.Atoi)); err != nil || got == nil || len(got) != 0 {
		t.Errorf("CollectResults of an empty stream = %#v, %v", got, err)
	}

	// The first failure stops the stream: later elements are never parsed.
	parsed := 0
	parse := func(s string) (int, error) { parsed++; return strconv.Atoi(s) }
	got, err = CollectResults(Of("1", "x", "3", "y").TryMap(parse))
	if _, ok := errors.AsType[*strconv.NumError](err); !ok || got != nil || parsed != 2 {
		t.Errorf("CollectResults with a failure = %v, %v after parsing %d", got, err, parsed)
	}

	// Without CollectResults every element gives a Result, so failures can be skipped.
	valid := Of("1", "x", "3").TryMap(strconv.Atoi).
		Filter(result.Result[int].IsOk).
		Map(result.Result[int].MustGet).
		ToSlice()
	if !slices.Equal(valid, []int{1, 3}) {
		t.Errorf("keeping the successes = %v", valid)
	}
	if got := Of("1", "x").TryMap(strconv.Atoi).Limit(1).Count(); got != 1 {
		t.Errorf("TryMap then Limit = %d", got)
	}
	mustPanic(t, func() { Of("1").TryMap[int](nil) })
}

func TestChannels(t *testing.T) {
	ctx := context.Background()
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	if got := FromChannel(ctx, ch).ToSlice(); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("FromChannel = %v", got)
	}

	// The stream ends when the context does, even though the channel stays open.
	open := make(chan int, 1)
	open <- 7
	cancelling, cancel := context.WithCancel(ctx)
	got := FromChannel(cancelling, open).Peek(func(int) { cancel() }).ToSlice()
	if !slices.Equal(got, []int{7}) {
		t.Errorf("FromChannel until the context ended = %v", got)
	}
	// A consumer that stops early leaves the rest in the channel.
	more := make(chan int, 2)
	more <- 1
	more <- 2
	if FromChannel(ctx, more).FindFirst().OrElse(0) != 1 || len(more) != 1 {
		t.Errorf("FindFirst on a channel took more than one value")
	}

	for _, buffer := range []int{0, 4} {
		out := Range(0, 5).ToChannel(ctx, buffer)
		var received []int
		for v := range out {
			received = append(received, v)
		}
		if !slices.Equal(received, []int{0, 1, 2, 3, 4}) {
			t.Errorf("ToChannel with buffer %d delivered %v", buffer, received)
		}
	}
	if got := FromChannel(ctx, Range(0, 100).Filter(IsEven).ToChannel(ctx, 0)).Limit(3).ToSlice(); !slices.Equal(got, []int{0, 2, 4}) {
		t.Errorf("a stream through a channel and back = %v", got)
	}
	mustPanic(t, func() { Of(1).ToChannel(ctx, -1) })

	// A receiver that gives up cancels the context, which stops the sending goroutine.
	before := runtime.NumGoroutine()
	abandoned, stop := context.WithCancel(ctx)
	infinite := Iterate(0, func(i int) int { return i + 1 }).ToChannel(abandoned, 0)
	<-infinite
	stop()
	for range infinite {
	}
	waitForGoroutines(t, before)
}

func TestParallelWithContext(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	var started atomic.Int32
	got := Iterate(0, func(i int) int { return i + 1 }).Parallel(4).WithContext(ctx).
		Map(func(i int) int {
			if started.Add(1) == 20 {
				cancel()
			}
			return i
		}).ToSlice()
	// An infinite source ended, with a gapless prefix of it as the result.
	if len(got) < 20 || len(got) > 40 {
		t.Errorf("%d elements came out, want about the 20 that had started", len(got))
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("element %d is %d: the result is not a prefix of the source", i, v)
		}
	}
	if n := int(started.Load()); n > len(got)+8 {
		t.Errorf("%d stage calls started for %d results", n, len(got))
	}
	waitForGoroutines(t, before)

	// A context that has already ended lets nothing through.
	ran := false
	if n := Range(0, 100).Parallel(4).WithContext(ctx).Peek(func(int) { ran = true }).Sequential().Count(); n != 0 || ran {
		t.Errorf("an ended context let %d elements through, ran = %v", n, ran)
	}
	// A context that stays alive changes nothing.
	alive := Range(0, 200).Parallel(4).WithContext(context.Background()).Map(func(i int) int { return i * 2 }).Filter(IsPositive).ToSlice()
	if len(alive) != 199 || alive[0] != 2 || alive[198] != 398 {
		t.Errorf("a live context changed the result: %d elements", len(alive))
	}
	waitForGoroutines(t, before)
}

// The context ends while every slot is taken, so the reader is waiting to start a task
// whose result the consumer is already waiting for.
func TestParallelContextEndsWhileWaitingForSlot(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 50 {
		ctx, cancel := context.WithCancel(context.Background())
		release := make(chan struct{})
		var once sync.Once
		done := make(chan []int, 1)
		go func() {
			done <- Range(0, 100).Parallel(2).WithContext(ctx).Map(func(i int) int {
				once.Do(func() {
					cancel()
					time.AfterFunc(time.Millisecond, func() { close(release) })
				})
				<-release
				return i
			}).ToSlice()
		}()
		select {
		case got := <-done:
			for i, v := range got {
				if v != i {
					t.Fatalf("element %d is %d", i, v)
				}
			}
			if len(got) > 4 {
				t.Fatalf("%d elements came out after the context ended", len(got))
			}
		case <-time.After(10 * time.Second):
			t.Fatal("the stream did not end after its context did")
		}
	}
	waitForGoroutines(t, before)
}

func TestParallelUnordered(t *testing.T) {
	before := runtime.NumGoroutine()
	got := Range(0, 500).Parallel(8).Unordered().Map(func(i int) int { return i * 2 }).Filter(func(v int) bool { return v%4 == 0 }).ToSlice()
	slices.Sort(got)
	if len(got) != 250 || got[0] != 0 || got[249] != 996 {
		t.Errorf("Unordered gave %d elements, want every one exactly once", len(got))
	}

	// A slow first element does not hold back the others.
	release := make(chan struct{})
	first := Range(0, 4).Parallel(4).Unordered().Map(func(i int) int {
		if i == 0 {
			<-release
		}
		return i
	}).Sequential().Peek(func(v int) {
		if v != 0 {
			select {
			case <-release:
			default:
				close(release)
			}
		}
	}).ToSlice()
	if len(first) != 4 || first[0] == 0 {
		t.Errorf("results came out as %v, want the slow element after a faster one", first)
	}

	// At most n stage calls run at once.
	var running, peak atomic.Int32
	Range(0, 200).Parallel(3).Unordered().ForEach(func(int) {
		n := running.Add(1)
		for {
			if old := peak.Load(); n <= old || peak.CompareAndSwap(old, n) {
				break
			}
		}
		runtime.Gosched()
		running.Add(-1)
	})
	if peak.Load() > 3 {
		t.Errorf("%d stage calls ran at once, want at most 3", peak.Load())
	}

	if got := Iterate(0, func(i int) int { return i + 1 }).Parallel(4).Unordered().Map(Identity[int]).Sequential().Limit(10).Count(); got != 10 {
		t.Errorf("Unordered over an infinite source then Limit = %d", got)
	}
	if got := Empty[int]().Parallel(2).Unordered().ToSlice(); len(got) != 0 {
		t.Errorf("Unordered of an empty stream = %v", got)
	}
	waitForGoroutines(t, before)
}

func TestParallelUnorderedPanicsAndContext(t *testing.T) {
	before := runtime.NumGoroutine()
	if v := mustPanic(t, func() {
		Range(0, 100).Parallel(4).Unordered().Map(func(i int) int {
			if i == 50 {
				panic("stage failed")
			}
			return i
		}).ToSlice()
	}); v != "stage failed" {
		t.Errorf("stage panic value = %v", v)
	}
	if v := mustPanic(t, func() {
		FromSeq(func(yield func(int) bool) {
			yield(1)
			panic("source failed")
		}).Parallel(2).Unordered().ToSlice()
	}); v != "source failed" {
		t.Errorf("source panic value = %v", v)
	}
	if v := mustPanic(t, func() {
		Of(1).Parallel(2).Unordered().Map(func(i int) int { runtime.Goexit(); return i }).ToSlice()
	}); v == nil {
		t.Errorf("Goexit in a stage was not reported")
	}
	if v := mustPanic(t, func() {
		Range(0, 100).Parallel(4).Unordered().Sequential().ForEach(func(int) { panic("consumer failed") })
	}); v != "consumer failed" {
		t.Errorf("consumer panic value = %v", v)
	}
	// A source that panics after the consumer has stopped must not block.
	Iterate(0, func(i int) int {
		if i > 5 {
			panic("late")
		}
		return i + 1
	}).Parallel(2).Unordered().Sequential().FindFirst()

	ctx, cancel := context.WithCancel(context.Background())
	var started atomic.Int32
	got := Iterate(0, func(i int) int { return i + 1 }).Parallel(4).Unordered().WithContext(ctx).Map(func(i int) int {
		if started.Add(1) == 20 {
			cancel()
		}
		return i
	}).ToSlice()
	if len(got) < 16 || len(got) > 40 {
		t.Errorf("%d elements came out of an unordered stream cut short at 20", len(got))
	}
	if n := Range(0, 100).Parallel(4).WithContext(ctx).Unordered().Sequential().Count(); n != 0 {
		t.Errorf("an ended context let %d elements through an unordered stream", n)
	}
	waitForGoroutines(t, before)
}

// The context ends while every slot of an unordered stream is taken.
func TestParallelUnorderedContextEndsWhileWaitingForSlot(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	var once sync.Once
	got := Range(0, 100).Parallel(2).Unordered().WithContext(ctx).Map(func(i int) int {
		once.Do(func() {
			cancel()
			time.AfterFunc(5*time.Millisecond, func() { close(release) })
		})
		<-release
		return i
	}).ToSlice()
	if len(got) > 4 {
		t.Errorf("%d elements came out after the context ended", len(got))
	}
	waitForGoroutines(t, before)
}
