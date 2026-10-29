package stream

import "testing"

func data(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = (i * 7919) % 10007
	}
	return s
}

var (
	ints10k = data(10_000)
	ints1k  = data(1_000)
	ints1m  = data(1_000_000)
	sink    any
)

func BenchmarkFilterMapReduce(b *testing.B) {
	for b.Loop() {
		sink = FromSlice(ints10k).
			Filter(func(i int) bool { return i%2 == 0 }).
			Map(func(i int) int { return i * 3 }).
			Reduce(func(a, b int) int { return a + b }).Get()
	}
}

func BenchmarkSorted1k(b *testing.B) {
	for b.Loop() {
		sink = FromSlice(ints1k).Sorted(func(a, b int) int { return a - b }).ToSlice()
	}
}

func BenchmarkDistinct10k(b *testing.B) {
	for b.Loop() {
		sink = Distinct(FromSlice(ints10k)).ToSlice()
	}
}

func BenchmarkFilterLimit1M(b *testing.B) {
	for b.Loop() {
		sink = FromSlice(ints1m).Filter(func(i int) bool { return i%3 == 0 }).Limit(10).ToSlice()
	}
}

func BenchmarkFilterFindFirst10k(b *testing.B) {
	for b.Loop() {
		sink = FromSlice(ints10k).Filter(func(i int) bool { return i > 5000 }).FindFirst().Get()
	}
}
