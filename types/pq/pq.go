// Package pq provides Queue, a priority queue backed by a binary heap.
//
// It is the counterpart of Java's PriorityQueue and a typed alternative to
// container/heap. The element that orders first by the comparator comes out first, so
// cmp.Compare gives a min-heap. Push and Pop take logarithmic time.
//
//	tasks := pq.New(func(a, b Task) int { return cmp.Compare(a.Deadline, b.Deadline) })
//	tasks.Push(t1, t2)
//	next, ok := tasks.Pop().Value()
//
// Elements that compare equal come out in an unspecified order. If insertion order
// matters, break ties with a sequence number in the comparator.
//
// A Queue must be created with New. It is not safe for concurrent use without
// external locking.
package pq

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

// Queue is a priority queue. See the package documentation.
type Queue[T any] struct {
	compare func(a, b T) int
	items   []T
}

// New returns a Queue ordered by compare and holding the given values. It panics if
// compare is nil. The values are heapified in linear time.
func New[T any](compare func(a, b T) int, values ...T) *Queue[T] {
	if compare == nil {
		panic("pq: New requires a non-nil comparator")
	}
	q := &Queue[T]{compare: compare, items: slices.Clone(values)}
	for i := len(q.items)/2 - 1; i >= 0; i-- {
		q.siftDown(i)
	}
	return q
}

// NewMin returns a Queue that pops the smallest value first.
func NewMin[T cmp.Ordered](values ...T) *Queue[T] {
	return New(cmp.Compare[T], values...)
}

// NewMax returns a Queue that pops the largest value first.
func NewMax[T cmp.Ordered](values ...T) *Queue[T] {
	return New(func(a, b T) int { return cmp.Compare(b, a) }, values...)
}

// Collect returns a Queue ordered by compare and holding the values of seq. It panics
// if compare is nil.
func Collect[T any](compare func(a, b T) int, seq iter.Seq[T]) *Queue[T] {
	return New(compare, slices.Collect(seq)...)
}

// PushPop adds v and then removes and returns the first element, which is v itself if
// nothing in the Queue orders before it. It does the work of a Push and a Pop in one
// pass, which is what keeping the n best of a long sequence needs:
//
//	if top.Len() < n {
//		top.Push(v)
//	} else {
//		top.PushPop(v) // drops the worst of the n+1
//	}
func (q *Queue[T]) PushPop(v T) T {
	if len(q.items) == 0 || q.compare(v, q.items[0]) <= 0 {
		return v
	}
	top := q.items[0]
	q.items[0] = v
	q.siftDown(0)
	return top
}

// RemoveFunc removes the elements that satisfy predicate and returns how many it
// removed. It takes linear time, whatever the number removed.
func (q *Queue[T]) RemoveFunc(predicate func(T) bool) int {
	before := len(q.items)
	q.items = slices.DeleteFunc(q.items, predicate)
	if len(q.items) != before {
		for i := len(q.items)/2 - 1; i >= 0; i-- {
			q.siftDown(i)
		}
	}
	return before - len(q.items)
}

// Push adds the given values.
func (q *Queue[T]) Push(values ...T) {
	for _, v := range values {
		q.items = append(q.items, v)
		q.siftUp(len(q.items) - 1)
	}
}

// Pop removes and returns the first element, or returns an empty Optional if the
// Queue is empty.
func (q *Queue[T]) Pop() optional.Optional[T] {
	if len(q.items) == 0 {
		return optional.Empty[T]()
	}
	top := q.items[0]
	last := len(q.items) - 1
	q.items[0] = q.items[last]
	var zero T
	q.items[last] = zero // Let the garbage collector reclaim what the slot refers to.
	q.items = q.items[:last]
	if last > 0 {
		q.siftDown(0)
	}
	return optional.Of(top)
}

// Peek returns the first element without removing it, or an empty Optional if the
// Queue is empty.
func (q *Queue[T]) Peek() optional.Optional[T] {
	if len(q.items) == 0 {
		return optional.Empty[T]()
	}
	return optional.Of(q.items[0])
}

// Len returns the number of elements in q.
func (q *Queue[T]) Len() int {
	return len(q.items)
}

// IsEmpty reports whether q has no elements.
func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}

// Clear removes every element from q and keeps its capacity for reuse.
func (q *Queue[T]) Clear() {
	clear(q.items)
	q.items = q.items[:0]
}

// Clone returns a copy of q with the same comparator that does not share storage with it.
func (q *Queue[T]) Clone() *Queue[T] {
	return &Queue[T]{compare: q.compare, items: slices.Clone(q.items)}
}

// All returns an iterator over the elements in no particular order, leaving them in
// the Queue. q must not be modified during iteration. Use Drain for priority order.
func (q *Queue[T]) All() iter.Seq[T] {
	return slices.Values(q.items)
}

// Drain returns an iterator that pops elements in priority order. Elements remain in
// the Queue if the loop stops early.
func (q *Queue[T]) Drain() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			v, ok := q.Pop().Value()
			if !ok || !yield(v) {
				return
			}
		}
	}
}

// String formats the elements in priority order, for example "Queue[1 2 3]". It sorts
// a copy, so it takes O(n log n) time and leaves q unchanged.
func (q *Queue[T]) String() string {
	var b strings.Builder
	b.WriteString("Queue[")
	first := true
	for v := range q.Clone().Drain() {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprint(&b, v)
	}
	b.WriteByte(']')
	return b.String()
}

func (q *Queue[T]) siftUp(i int) {
	v := q.items[i]
	for i > 0 {
		parent := (i - 1) / 2
		if q.compare(v, q.items[parent]) >= 0 {
			break
		}
		q.items[i] = q.items[parent]
		i = parent
	}
	q.items[i] = v
}

func (q *Queue[T]) siftDown(i int) {
	v := q.items[i]
	n := len(q.items)
	for {
		child := 2*i + 1
		if child >= n {
			break
		}
		if right := child + 1; right < n && q.compare(q.items[right], q.items[child]) < 0 {
			child = right
		}
		if q.compare(q.items[child], v) >= 0 {
			break
		}
		q.items[i] = q.items[child]
		i = child
	}
	q.items[i] = v
}
