// Package deque provides Deque, a double-ended queue backed by a growable ring buffer.
//
// A Deque works as a FIFO queue (PushBack and PopFront) or a LIFO stack (PushBack and
// PopBack). Adding or removing at either end takes amortized constant time. It is the
// counterpart of Java's ArrayDeque.
//
// The zero value is an empty Deque ready to use. A Deque must not be copied after
// first use, and it is not safe for concurrent use without external locking.
package deque

import (
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

const minCapacity = 8

// Deque is a double-ended queue. See the package documentation.
type Deque[T any] struct {
	buf  []T
	head int
	n    int
}

// Of returns a Deque holding the given values, with the first value at the front.
func Of[T any](values ...T) *Deque[T] {
	d := &Deque[T]{}
	d.PushBack(values...)
	return d
}

// Collect returns a Deque holding the values of seq, with the first value at the front.
func Collect[T any](seq iter.Seq[T]) *Deque[T] {
	d := &Deque[T]{}
	for v := range seq {
		d.PushBack(v)
	}
	return d
}

// PushBack adds the given values to the back, in order.
func (d *Deque[T]) PushBack(values ...T) {
	d.reserve(len(values))
	for _, v := range values {
		d.buf[d.index(d.n)] = v
		d.n++
	}
}

// PushFront adds the given values to the front, one at a time, so the last value ends
// up at the very front.
func (d *Deque[T]) PushFront(values ...T) {
	d.reserve(len(values))
	for _, v := range values {
		d.head = d.index(len(d.buf) - 1)
		d.buf[d.head] = v
		d.n++
	}
}

// PopFront removes and returns the value at the front, or returns an empty Optional
// if the Deque is empty.
func (d *Deque[T]) PopFront() optional.Optional[T] {
	if d.n == 0 {
		return optional.Empty[T]()
	}
	v := d.buf[d.head]
	var zero T
	d.buf[d.head] = zero // Let the garbage collector reclaim what v refers to.
	d.head = d.index(1)
	d.n--
	return optional.Of(v)
}

// PopBack removes and returns the value at the back, or returns an empty Optional if
// the Deque is empty.
func (d *Deque[T]) PopBack() optional.Optional[T] {
	if d.n == 0 {
		return optional.Empty[T]()
	}
	i := d.index(d.n - 1)
	v := d.buf[i]
	var zero T
	d.buf[i] = zero // Let the garbage collector reclaim what v refers to.
	d.n--
	return optional.Of(v)
}

// Front returns the value at the front without removing it, or an empty Optional if
// the Deque is empty.
func (d *Deque[T]) Front() optional.Optional[T] {
	if d.n == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(d.buf[d.head])
}

// Back returns the value at the back without removing it, or an empty Optional if the
// Deque is empty.
func (d *Deque[T]) Back() optional.Optional[T] {
	if d.n == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(d.buf[d.index(d.n-1)])
}

// At returns the value i positions from the front. It panics if i is out of range.
func (d *Deque[T]) At(i int) T {
	if i < 0 || i >= d.n {
		panic(fmt.Sprintf("deque: index %d out of range with length %d", i, d.n))
	}
	return d.buf[d.index(i)]
}

// Len returns the number of values in d.
func (d *Deque[T]) Len() int {
	return d.n
}

// IsEmpty reports whether d has no values.
func (d *Deque[T]) IsEmpty() bool {
	return d.n == 0
}

// Clear removes every value from d and keeps its capacity for reuse.
func (d *Deque[T]) Clear() {
	clear(d.buf)
	d.head, d.n = 0, 0
}

// Clone returns a copy of d that does not share storage with it.
func (d *Deque[T]) Clone() *Deque[T] {
	c := &Deque[T]{}
	c.reserve(d.n)
	for v := range d.All() {
		c.PushBack(v)
	}
	return c
}

// All returns an iterator over the values from front to back. d must not be modified
// during iteration.
func (d *Deque[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range d.n {
			if !yield(d.buf[d.index(i)]) {
				return
			}
		}
	}
}

// Backward returns an iterator over the values from back to front. d must not be
// modified during iteration.
func (d *Deque[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := d.n - 1; i >= 0; i-- {
			if !yield(d.buf[d.index(i)]) {
				return
			}
		}
	}
}

// String formats the values from front to back, for example "Deque[1 2 3]".
func (d *Deque[T]) String() string {
	var b strings.Builder
	b.WriteString("Deque[")
	for i := range d.n {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprint(&b, d.buf[d.index(i)])
	}
	b.WriteByte(']')
	return b.String()
}

// index returns the buffer position i places after the front, wrapping around. It
// requires 0 <= i < len(d.buf), which also rules out an empty buffer.
func (d *Deque[T]) index(i int) int {
	i += d.head
	if i >= len(d.buf) {
		i -= len(d.buf)
	}
	return i
}

// reserve makes room for extra more values, moving the values to the start of a new
// buffer if they do not fit.
func (d *Deque[T]) reserve(extra int) {
	if d.n+extra <= len(d.buf) {
		return
	}
	capacity := max(len(d.buf)*2, minCapacity)
	for capacity < d.n+extra {
		capacity *= 2
	}
	buf := make([]T, capacity)
	if d.n > 0 {
		end := min(d.head+d.n, len(d.buf))
		copied := copy(buf, d.buf[d.head:end])
		copy(buf[copied:], d.buf[:d.n-copied])
	}
	d.buf, d.head = buf, 0
}
