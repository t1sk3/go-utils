package stream

import (
	"cmp"
	"iter"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/constraints"
	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
	"github.com/t1sk3/go-utils/types/set"
)

// This file holds the operations that cannot be methods on Stream: those that
// constrain T, and Enumerate, Chunk, Window and Zip. See methods.go for why.

// Distinct returns a Stream of the elements of s with duplicates removed, keeping
// the first occurrence of each.
func Distinct[T comparable](s Stream[T]) Stream[T] {
	return s.DistinctBy(func(v T) T { return v })
}

// Sum returns the sum of the elements of s, or zero for an empty stream.
func Sum[T constraints.Number](s Stream[T]) T {
	var total T
	for v := range s.All() {
		total += v
	}
	return total
}

// Average returns the arithmetic mean of the elements of s, or an empty Optional if s
// is empty. The elements are summed as float64, so very large integers may lose
// precision.
func Average[T constraints.Number](s Stream[T]) optional.Optional[float64] {
	var total float64
	n := 0
	for v := range s.All() {
		total += float64(v)
		n++
	}
	if n == 0 {
		return optional.Empty[float64]()
	}
	return optional.Of(total / float64(n))
}

// Summary holds the statistics computed by Summarize.
type Summary[T constraints.Number] struct {
	// Count is the number of elements. The other fields are zero when it is zero.
	Count int
	// Sum is the total, which can overflow T just as it does with the Sum function.
	Sum T
	// Min and Max are the smallest and largest elements. They follow cmp.Compare, as the
	// Min and Max functions do, so NaN counts as smaller than any other float.
	Min, Max T
	// Average is the arithmetic mean. It is computed from a separate float64 total, so
	// small integer types cannot wrap around, but very large integers may lose precision.
	Average float64
}

// Summarize returns the count, sum, minimum, maximum and average of the elements of s
// in a single pass, which suits streams that can only be consumed once.
func Summarize[T constraints.Number](s Stream[T]) Summary[T] {
	var sum Summary[T]
	var total float64
	for v := range s.All() {
		if sum.Count == 0 {
			sum.Min, sum.Max = v, v
		} else {
			if cmp.Compare(v, sum.Min) < 0 {
				sum.Min = v
			}
			if cmp.Compare(v, sum.Max) > 0 {
				sum.Max = v
			}
		}
		sum.Count++
		sum.Sum += v
		total += float64(v)
	}
	if sum.Count > 0 {
		sum.Average = total / float64(sum.Count)
	}
	return sum
}

// Sort returns a Stream of the elements of s in ascending natural order. Use
// Stream.Sorted for a custom ordering.
func Sort[T cmp.Ordered](s Stream[T]) Stream[T] {
	return s.sortedWith(slices.Sort[[]T])
}

// Min returns the smallest element of s, or an empty Optional if s is empty.
func Min[T cmp.Ordered](s Stream[T]) optional.Optional[T] {
	return s.Min(cmp.Compare[T])
}

// Max returns the largest element of s, or an empty Optional if s is empty.
func Max[T cmp.Ordered](s Stream[T]) optional.Optional[T] {
	return s.Max(cmp.Compare[T])
}

// Identity returns v unchanged. It is useful as a key or value function when the
// element itself is what you want:
//
//	lengths := words.ToMap(stream.Identity, func(w string) int { return len(w) })
func Identity[T any](v T) T {
	return v
}

// ToSet collects the elements of s into a set.Set.
func ToSet[T comparable](s Stream[T]) set.Set[T] {
	return set.Collect(s.All())
}

// ToSeq2 returns a stream of pairs as an iter.Seq2, for use with two-value range
// loops or functions such as maps.Collect:
//
//	byID := maps.Collect(stream.ToSeq2(stream.Zip(ids, users)))
func ToSeq2[K, V any](s Stream[pair.Pair[K, V]]) iter.Seq2[K, V] {
	src := s.All()
	return func(yield func(K, V) bool) {
		for p := range src {
			if !yield(p.First, p.Second) {
				return
			}
		}
	}
}

// Zip returns a Stream that pairs the elements of a and b by position. It ends when
// either stream ends, so zipping with an infinite stream is fine.
//
// When b comes straight from FromSlice or Of, Zip indexes it directly; otherwise it
// has to step b with iter.Pull, which is roughly ten times slower per element. If
// only one side is a plain slice, pass it as b.
func Zip[A, B any](a Stream[A], b Stream[B]) Stream[pair.Pair[A, B]] {
	size := -1
	if a.size >= 0 && b.size >= 0 {
		size = min(a.size, b.size)
	}
	srcA := a.All()
	if b.seq == nil {
		// b is a plain slice: index it instead of starting a coroutine with iter.Pull.
		bs := b.slice
		return Stream[pair.Pair[A, B]]{size: size, seq: func(yield func(pair.Pair[A, B]) bool) {
			if len(bs) == 0 {
				return
			}
			i := 0
			for va := range srcA {
				if !yield(pair.Of(va, bs[i])) {
					return
				}
				if i++; i == len(bs) {
					return
				}
			}
		}}
	}
	srcB := b.seq
	return Stream[pair.Pair[A, B]]{size: size, seq: func(yield func(pair.Pair[A, B]) bool) {
		nextB, stop := iter.Pull(srcB)
		defer stop()
		for va := range srcA {
			vb, ok := nextB()
			if !ok || !yield(pair.Of(va, vb)) {
				return
			}
		}
	}}
}

// Enumerate returns a Stream that pairs each element of s with its index, starting at 0.
func Enumerate[T any](s Stream[T]) Stream[pair.Pair[int, T]] {
	src := s.All()
	return Stream[pair.Pair[int, T]]{size: s.size, seq: func(yield func(pair.Pair[int, T]) bool) {
		i := 0
		for v := range src {
			if !yield(pair.Of(i, v)) {
				return
			}
			i++
		}
	}}
}

// Chunk returns a Stream of consecutive, non-overlapping slices of n elements of s.
// The last chunk is shorter if the elements do not divide evenly. Each chunk is a
// new slice that the caller may keep or modify. Chunk panics if n is not positive.
//
//	stream.Chunk(stream.Range(0, 5), 2) // [0 1] [2 3] [4]
func Chunk[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		panic("stream: Chunk requires a positive size")
	}
	size := -1
	if s.size >= 0 {
		size = s.size / n
		if s.size%n != 0 {
			size++
		}
	}
	src := s.All()
	capacity := capFor(n, s.size)
	return Stream[[]T]{size: size, seq: func(yield func([]T) bool) {
		var buf []T
		for v := range src {
			if buf == nil {
				buf = make([]T, 0, capacity)
			}
			buf = append(buf, v)
			if len(buf) == n {
				if !yield(buf) {
					return
				}
				buf = nil
			}
		}
		if len(buf) > 0 {
			yield(buf)
		}
	}}
}

// Window returns a Stream of every run of n consecutive elements of s, advancing one
// element at a time. A stream with fewer than n elements produces no windows. Each
// window is a new slice that the caller may keep or modify. Window panics if n is
// not positive.
//
//	stream.Window(stream.Range(0, 4), 2) // [0 1] [1 2] [2 3]
func Window[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		panic("stream: Window requires a positive size")
	}
	size := -1
	if s.size >= 0 {
		size = max(s.size-n+1, 0)
	}
	src := s.All()
	capacity := capFor(n, s.size)
	return Stream[[]T]{size: size, seq: func(yield func([]T) bool) {
		buf := make([]T, 0, capacity)
		for v := range src {
			if len(buf) == n {
				copy(buf, buf[1:])
				buf = buf[:n-1]
			}
			buf = append(buf, v)
			if len(buf) == n && !yield(slices.Clone(buf)) {
				return
			}
		}
	}}
}

// capFor returns the capacity to allocate for a buffer of up to n elements drawn
// from a stream of the given size, which may be -1 if unknown.
func capFor(n, size int) int {
	if size >= 0 {
		return min(n, size)
	}
	return n
}

// Join concatenates the elements of s, placing sep between them.
func Join[T ~string](s Stream[T], sep string) string {
	var b strings.Builder
	first := true
	for v := range s.All() {
		if !first {
			b.WriteString(sep)
		}
		first = false
		b.WriteString(string(v))
	}
	return b.String()
}
