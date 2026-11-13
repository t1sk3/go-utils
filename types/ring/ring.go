// Package ring provides Ring, a fixed-size buffer that overwrites its oldest value
// when full.
//
// It keeps the last n of something without ever growing: recent log lines, the latest
// samples of a metric, an undo history with a limit.
//
//	recent := ring.New[string](100)
//	recent.Push(line) // the 101st line pushes the first one out
//
// A Ring must be created with New. It must not be copied after first use, and it is not
// safe for concurrent use without external locking.
package ring

import (
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

// Ring is a fixed-size buffer of the most recent values. See the package documentation.
type Ring[T any] struct {
	buf  []T
	head int // position of the oldest value
	n    int
}

// New returns an empty Ring that holds up to capacity values. It panics if capacity is
// not positive.
func New[T any](capacity int) *Ring[T] {
	if capacity <= 0 {
		panic("ring: New requires a positive capacity")
	}
	return &Ring[T]{buf: make([]T, capacity)}
}

// Push adds v as the newest value. If the Ring is full, the oldest value makes room
// and is returned.
func (r *Ring[T]) Push(v T) optional.Optional[T] {
	if r.n < len(r.buf) {
		r.buf[r.index(r.n)] = v
		r.n++
		return optional.Empty[T]()
	}
	evicted := r.buf[r.head]
	r.buf[r.head] = v
	r.head = r.index(1)
	return optional.Of(evicted)
}

// Pop removes and returns the oldest value, or returns an empty Optional if the Ring is
// empty.
func (r *Ring[T]) Pop() optional.Optional[T] {
	if r.n == 0 {
		return optional.Empty[T]()
	}
	v := r.buf[r.head]
	var zero T
	r.buf[r.head] = zero // Let the garbage collector reclaim what v refers to.
	r.head = r.index(1)
	r.n--
	return optional.Of(v)
}

// Oldest returns the oldest value without removing it, or an empty Optional if the Ring
// is empty.
func (r *Ring[T]) Oldest() optional.Optional[T] {
	if r.n == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(r.buf[r.head])
}

// Newest returns the newest value without removing it, or an empty Optional if the Ring
// is empty.
func (r *Ring[T]) Newest() optional.Optional[T] {
	if r.n == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(r.buf[r.index(r.n-1)])
}

// At returns the value i positions from the oldest. It panics if i is out of range.
func (r *Ring[T]) At(i int) T {
	if i < 0 || i >= r.n {
		panic(fmt.Sprintf("ring: index %d out of range with length %d", i, r.n))
	}
	return r.buf[r.index(i)]
}

// Len returns the number of values in r.
func (r *Ring[T]) Len() int {
	return r.n
}

// Cap returns the number of values r can hold.
func (r *Ring[T]) Cap() int {
	return len(r.buf)
}

// IsEmpty reports whether r has no values.
func (r *Ring[T]) IsEmpty() bool {
	return r.n == 0
}

// IsFull reports whether the next Push will overwrite the oldest value.
func (r *Ring[T]) IsFull() bool {
	return r.n == len(r.buf)
}

// Clear removes every value from r.
func (r *Ring[T]) Clear() {
	clear(r.buf)
	r.head, r.n = 0, 0
}

// ToSlice returns the values from oldest to newest in a new slice. The result is never
// nil.
func (r *Ring[T]) ToSlice() []T {
	out := make([]T, 0, r.n)
	for i := range r.n {
		out = append(out, r.buf[r.index(i)])
	}
	return out
}

// All returns an iterator over the values from oldest to newest. r must not be
// modified during iteration.
func (r *Ring[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range r.n {
			if !yield(r.buf[r.index(i)]) {
				return
			}
		}
	}
}

// Backward returns an iterator over the values from newest to oldest. r must not be
// modified during iteration.
func (r *Ring[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := r.n - 1; i >= 0; i-- {
			if !yield(r.buf[r.index(i)]) {
				return
			}
		}
	}
}

// String formats the values from oldest to newest, for example "Ring[1 2 3]".
func (r *Ring[T]) String() string {
	var b strings.Builder
	b.WriteString("Ring[")
	for i := range r.n {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, r.buf[r.index(i)])
	}
	b.WriteByte(']')
	return b.String()
}

// index returns the buffer position i places after the oldest value, wrapping around.
// It requires 0 <= i <= len(r.buf).
func (r *Ring[T]) index(i int) int {
	i += r.head
	if i >= len(r.buf) {
		i -= len(r.buf)
	}
	return i
}
