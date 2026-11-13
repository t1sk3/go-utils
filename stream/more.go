package stream

import (
	"cmp"
	"context"
	"iter"
	"slices"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
	"github.com/t1sk3/go-utils/types/pq"
	"github.com/t1sk3/go-utils/types/result"
)

// Reverse returns a Stream of the elements in the opposite order. It has to read every
// element before it can produce the first, so bound an infinite stream first.
func (s Stream[T]) Reverse() Stream[T] {
	return Stream[T]{size: s.size, seq: func(yield func(T) bool) {
		buf := s.ToSlice()
		for i := len(buf) - 1; i >= 0; i-- {
			if !yield(buf[i]) {
				return
			}
		}
	}}
}

// Intersperse returns a Stream with sep placed between every two elements:
//
//	stream.Of("a", "b", "c").Intersperse("-") // a - b - c
func (s Stream[T]) Intersperse(sep T) Stream[T] {
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		first := true
		for v := range src {
			if !first && !yield(sep) {
				return
			}
			first = false
			if !yield(v) {
				return
			}
		}
	}}
}

// Cycle returns a Stream that repeats the elements of s forever, in order. It is empty
// if s is empty. The elements are read from s once and remembered, so s may be a stream
// that can only be consumed once, but must be finite. Bound the result with Limit or
// TakeWhile.
//
//	stream.Of("red", "green").Cycle().Limit(5) // red green red green red
func (s Stream[T]) Cycle() Stream[T] {
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		var seen []T
		for v := range src {
			seen = append(seen, v)
			if !yield(v) {
				return
			}
		}
		for len(seen) > 0 {
			for _, v := range seen {
				if !yield(v) {
					return
				}
			}
		}
	}}
}

// TakeLast returns a Stream of at most the last n elements, in their original order. It
// has to read every element, but remembers only n of them at a time. TakeLast panics if
// n is negative.
func (s Stream[T]) TakeLast(n int) Stream[T] {
	if n < 0 {
		panic("stream: TakeLast requires a non-negative count")
	}
	if s.seq == nil {
		return FromSlice(s.slice[max(len(s.slice)-n, 0):])
	}
	size := -1
	if s.size >= 0 {
		size = min(n, s.size)
	}
	src := s.seq
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		if n == 0 {
			return
		}
		// buf is a ring: once it holds n elements, next is where the oldest one sits.
		buf := make([]T, 0, capFor(n, s.size))
		next := 0
		for v := range src {
			if len(buf) < n {
				buf = append(buf, v)
			} else {
				buf[next] = v
				next = (next + 1) % n
			}
		}
		for i := range buf {
			if !yield(buf[(next+i)%len(buf)]) {
				return
			}
		}
	}}
}

// Top returns a Stream of the n elements that come first in the order defined by
// comparator, in that order. It gives what Sorted(comparator).Limit(n) gives, but
// remembers only n elements instead of sorting all of them, which is what makes "the
// ten largest of a million" cheap:
//
//	largest := sizes.Top(10, stream.Reversed(cmp.Compare[int]))
//
// Unlike Sorted it is not stable: among elements that compare equal, which ones are
// kept and in what order is unspecified. Top panics if n is negative.
func (s Stream[T]) Top(n int, comparator func(a, b T) int) Stream[T] {
	checkFunc(comparator == nil, "Top")
	if n < 0 {
		panic("stream: Top requires a non-negative count")
	}
	size := -1
	if s.size >= 0 {
		size = min(n, s.size)
	}
	src := s.All()
	return Stream[T]{size: size, seq: func(yield func(T) bool) {
		if n == 0 {
			return
		}
		// The heap keeps the n best seen so far with the worst of them on top, where
		// the next element can displace it.
		worstFirst := pq.New(func(a, b T) int { return comparator(b, a) })
		for v := range src {
			if worstFirst.Len() < n {
				worstFirst.Push(v)
			} else {
				worstFirst.PushPop(v)
			}
		}
		best := slices.Collect(worstFirst.Drain())
		for i := len(best) - 1; i >= 0; i-- {
			if !yield(best[i]) {
				return
			}
		}
	}}
}

// MinBy returns the element for which key is smallest, or an empty Optional if the
// stream is empty. If several elements share the smallest key, the first is returned.
// key runs once per element.
//
//	youngest := users.MinBy(User.Age)
func (s Stream[T]) MinBy[K cmp.Ordered](key func(T) K) optional.Optional[T] {
	checkFunc(key == nil, "MinBy")
	return bestByKey(s, key, func(c int) bool { return c < 0 })
}

// MaxBy returns the element for which key is largest, or an empty Optional if the
// stream is empty. If several elements share the largest key, the first is returned.
// key runs once per element.
func (s Stream[T]) MaxBy[K cmp.Ordered](key func(T) K) optional.Optional[T] {
	checkFunc(key == nil, "MaxBy")
	return bestByKey(s, key, func(c int) bool { return c > 0 })
}

func bestByKey[T any, K cmp.Ordered](s Stream[T], key func(T) K, better func(int) bool) optional.Optional[T] {
	var best T
	var bestKey K
	found := false
	for v := range s.All() {
		if k := key(v); !found || better(cmp.Compare(k, bestKey)) {
			best, bestKey, found = v, k, true
		}
	}
	return optional.OfOk(best, found)
}

// ZipWith returns a Stream of f applied to the elements of s and b by position. It ends
// when either stream ends. It is Zip without the intermediate Pair:
//
//	totals := prices.ZipWith(quantities, func(p float64, q int) float64 { return p * float64(q) })
//
// What Zip says about which stream to pass as b, and about the element of s that may
// be read and dropped, holds here too.
func (s Stream[T]) ZipWith[B, R any](b Stream[B], f func(T, B) R) Stream[R] {
	checkFunc(f == nil, "ZipWith")
	size := -1
	if s.size >= 0 && b.size >= 0 {
		size = min(s.size, b.size)
	}
	srcA := s.All()
	return Stream[R]{size: size, seq: func(yield func(R) bool) {
		if b.seq == nil {
			bs := b.slice
			if len(bs) == 0 {
				return
			}
			i := 0
			for va := range srcA {
				if !yield(f(va, bs[i])) {
					return
				}
				if i++; i == len(bs) {
					return
				}
			}
			return
		}
		nextB, stop := iter.Pull(b.seq)
		defer stop()
		for va := range srcA {
			vb, ok := nextB()
			if !ok || !yield(f(va, vb)) {
				return
			}
		}
	}}
}

// Unzip collects a stream of pairs into a slice of the first values and a slice of the
// second values, in order. It is the inverse of Zip. Neither result is nil.
func Unzip[A, B any](s Stream[pair.Pair[A, B]]) ([]A, []B) {
	as, bs := make([]A, 0, sizeHint(s.size)), make([]B, 0, sizeHint(s.size))
	for p := range s.All() {
		as = append(as, p.First)
		bs = append(bs, p.Second)
	}
	return as, bs
}

// Flatten returns a Stream of the elements of each stream in s, in turn. It is Concat
// for streams that are themselves produced lazily, and FlatMap with nothing to map.
func Flatten[T any](s Stream[Stream[T]]) Stream[T] {
	src := s.All()
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for inner := range src {
			for v := range inner.All() {
				if !yield(v) {
					return
				}
			}
		}
	}}
}

// TryMap returns a Stream of the outcome of applying mapper, which can fail, to each
// element. Every element produces a Result, so a failure does not stop the stream by
// itself; pass the stream to CollectResults to stop at the first one, or handle each
// Result as it comes:
//
//	numbers, err := stream.CollectResults(lines.TryMap(strconv.Atoi))
func (s Stream[T]) TryMap[R any](mapper func(T) (R, error)) Stream[result.Result[R]] {
	checkFunc(mapper == nil, "TryMap")
	src := s.All()
	return Stream[result.Result[R]]{size: s.size, seq: func(yield func(result.Result[R]) bool) {
		for v := range src {
			if !yield(result.Of(mapper(v))) {
				return
			}
		}
	}}
}

// CollectResults collects the values of a stream of Results into a slice, or returns
// the first error. It stops reading at that error, so the stages before it do not run
// for the elements after it. The slice is never nil on success and nil on failure.
func CollectResults[T any](s Stream[result.Result[T]]) ([]T, error) {
	out := make([]T, 0, sizeHint(s.size))
	for r := range s.All() {
		v, err := r.Get()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// FromChannel returns a Stream of the values received from ch. It ends when ch is
// closed or ctx ends, whichever comes first; check ctx.Err() afterwards to tell which.
// Each value is received once, so the stream can only be consumed once.
func FromChannel[T any](ctx context.Context, ch <-chan T) Stream[T] {
	return Stream[T]{size: -1, seq: func(yield func(T) bool) {
		for {
			select {
			case v, ok := <-ch:
				if !ok || !yield(v) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}}
}

// ToChannel runs the stream on a new goroutine and returns a channel that receives its
// elements and is closed after the last one. buffer is the capacity of the channel.
//
// The goroutine stops early when ctx ends, which is the only way to stop it if the
// receiver gives up before the channel is closed: a receiver that might do so must
// cancel ctx, or the goroutine stays blocked on its next send.
//
// The stream's stages run on that goroutine, so a panic in one of them cannot be
// recovered by the caller and ends the program; a stage that can panic should recover
// itself. ToChannel panics if buffer is negative.
func (s Stream[T]) ToChannel(ctx context.Context, buffer int) <-chan T {
	if buffer < 0 {
		panic("stream: ToChannel requires a non-negative buffer")
	}
	ch := make(chan T, buffer)
	src := s.All()
	go func() {
		defer close(ch)
		for v := range src {
			select {
			case ch <- v:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
