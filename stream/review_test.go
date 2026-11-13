package stream

import (
	"iter"
	"math"
	"slices"
	"testing"
)

// These operations size a buffer from the stream's size or the caller's n, which say
// how many elements may come, not how many will. A huge value must not turn into a huge
// allocation.
func TestBuffersAreNotSizedFromHints(t *testing.T) {
	three := FromSeq(iter.Seq[int](func(yield func(int) bool) { _ = yield(1) && yield(2) && yield(3) }))
	huge := Range(0, math.MaxInt)
	cases := map[string]func(){
		"Distinct then Limit":   func() { Distinct(huge).Limit(3).ToSlice() },
		"DistinctBy then Limit": func() { huge.DistinctBy(Identity[int]).Limit(3).ToSlice() },
		"Chunk larger than stream": func() {
			if got := three.Chunk(math.MaxInt).ToSlice(); len(got) != 1 || !slices.Equal(got[0], []int{1, 2, 3}) {
				t.Errorf("Chunk = %v", got)
			}
		},
		"Chunk of a sized stream": func() { huge.Chunk(math.MaxInt).Limit(0).ToSlice() },
		"Window larger than stream": func() {
			if got := three.Window(math.MaxInt).Count(); got != 0 {
				t.Errorf("Window gave %d windows", got)
			}
		},
		"ToMap with few keys": func() {
			m := Range(0, 5_000_000).ToMap(func(v int) int { return v % 3 }, Identity[int])
			if len(m) != 3 {
				t.Errorf("ToMap has %d keys", len(m))
			}
		},
		"ToMapMerge with few keys": func() {
			m := Range(0, 5_000_000).ToMapMerge(func(v int) int { return v % 3 }, Identity[int],
				func(a, b int) int { return a + b })
			if len(m) != 3 {
				t.Errorf("ToMapMerge has %d keys", len(m))
			}
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			const limit = 4 << 20
			if got := testing.AllocsPerRun(1, run); got > 1000 {
				t.Errorf("%v allocations", got)
			}
			if b := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					run()
				}
			}).AllocedBytesPerOp(); b > limit {
				t.Errorf("allocated %d bytes per run, want at most %d", b, limit)
			}
		})
	}
}

func TestChunkAndWindowStillCorrectPastPrealloc(t *testing.T) {
	n := maxPrealloc*2 + 5
	src := FromSeq(Range(0, n*2).All())
	chunks := src.Chunk(n).ToSlice()
	if len(chunks) != 2 || len(chunks[0]) != n || chunks[1][0] != n {
		t.Errorf("Chunk(%d) gave %d chunks", n, len(chunks))
	}
	windows := src.Window(n).ToSlice()
	if len(windows) != n+1 || len(windows[0]) != n || windows[n][0] != n {
		t.Errorf("Window(%d) gave %d windows", n, len(windows))
	}
	if got := Distinct(Range(0, n).Map(func(v int) int { return v / 2 })).Count(); got != (n+1)/2 {
		t.Errorf("Distinct kept %d", got)
	}
}

// The combinators keep their arguments, so changing the slice afterwards must not
// change, or break, what they return.
func TestVariadicArgumentsAreCopied(t *testing.T) {
	preds := []func(int) bool{IsEven[int], IsPositive[int]}
	and, or := And(preds...), Or(preds...)
	preds[0], preds[1] = nil, nil
	if !and(2) || and(3) || !or(3) || or(-1) {
		t.Errorf("And or Or changed with the caller's slice")
	}

	streams := []Stream[int]{Of(1), Of(2)}
	both := Concat(streams...)
	streams[0] = Of(9)
	if got := both.ToSlice(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("Concat = %v after the caller's slice changed", got)
	}
}

// Zip reads the receiver first, so when the other side is an iterator that ends first,
// exactly one element of the receiver is read and dropped. This pins the documented
// behaviour for both ways of stepping b.
func TestZipReadsFromReceiver(t *testing.T) {
	for name, b := range map[string]Stream[string]{
		"slice":    Of("x", "y"),
		"iterator": FromSeq(slices.Values([]string{"x", "y"})),
	} {
		pulled := 0
		a := Generate(func() int { pulled++; return pulled })
		if got := a.Zip(b).Count(); got != 2 {
			t.Errorf("%s: %d pairs", name, got)
		}
		if want := map[string]int{"slice": 2, "iterator": 3}[name]; pulled != want {
			t.Errorf("%s: read %d elements of the receiver, want %d", name, pulled, want)
		}
	}
}

func TestDistinctKeepsEveryNaN(t *testing.T) {
	got := Distinct(Of(math.NaN(), math.NaN(), 1, 1)).ToSlice()
	if len(got) != 3 || !math.IsNaN(got[0]) || !math.IsNaN(got[1]) || got[2] != 1 {
		t.Errorf("Distinct = %v", got)
	}
}
