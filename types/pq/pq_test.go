package pq

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestNewRequiresComparator(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(nil) did not panic")
		}
	}()
	New[int](nil)
}

func TestEmpty(t *testing.T) {
	q := New(cmp.Compare[int])
	if !q.IsEmpty() || q.Len() != 0 || q.Pop().IsPresent() || q.Peek().IsPresent() {
		t.Error("a new Queue is not empty")
	}
}

func TestNewDoesNotAliasOrModifyValues(t *testing.T) {
	values := []int{3, 1, 2}
	q := New(cmp.Compare[int], values...)
	q.Pop()
	if !slices.Equal(values, []int{3, 1, 2}) {
		t.Errorf("New changed the caller's slice: %v", values)
	}
}

func TestOrderAgainstSort(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for _, n := range []int{1, 2, 3, 10, 1000} {
		values := make([]int, n)
		for i := range values {
			values[i] = rng.IntN(n) // Include duplicates.
		}
		want := slices.Sorted(slices.Values(values))

		heapified := New(cmp.Compare[int], values...)
		if got := slices.Collect(heapified.Drain()); !slices.Equal(got, want) {
			t.Errorf("heapified %d values out of order", n)
		}

		pushed := New(cmp.Compare[int])
		pushed.Push(values...)
		if got := slices.Collect(pushed.Drain()); !slices.Equal(got, want) {
			t.Errorf("pushed %d values out of order", n)
		}
	}
}

func TestMixedOperations(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	q := New(cmp.Compare[int])
	var model []int
	for range 20_000 {
		if rng.IntN(3) != 0 || len(model) == 0 {
			v := rng.IntN(1000)
			q.Push(v)
			i, _ := slices.BinarySearch(model, v)
			model = slices.Insert(model, i, v)
		} else {
			if got := q.Peek().MustGet(); got != model[0] {
				t.Fatalf("Peek = %d, want %d", got, model[0])
			}
			if got := q.Pop().MustGet(); got != model[0] {
				t.Fatalf("Pop = %d, want %d", got, model[0])
			}
			model = model[1:]
		}
	}
	if q.Len() != len(model) {
		t.Errorf("Len = %d, want %d", q.Len(), len(model))
	}
}

func TestComparatorDirection(t *testing.T) {
	maxHeap := New(func(a, b int) int { return cmp.Compare(b, a) }, 1, 3, 2)
	if got := maxHeap.Pop().MustGet(); got != 3 {
		t.Errorf("Pop = %d, want 3", got)
	}
}

func TestPopClearsSlot(t *testing.T) {
	a, b := 1, 2
	q := New(func(x, y *int) int { return cmp.Compare(*x, *y) }, &a, &b)
	q.Pop()
	if got := q.items[:cap(q.items)][1]; got != nil {
		t.Error("the popped slot still refers to a value")
	}
}

func TestDrainStopsEarly(t *testing.T) {
	q := New(cmp.Compare[int], 1, 2, 3)
	for v := range q.Drain() {
		if v == 1 {
			break
		}
	}
	if q.Len() != 2 || q.Peek().MustGet() != 2 {
		t.Errorf("after stopping early: len %d, top %v", q.Len(), q.Peek())
	}
}

func TestClone(t *testing.T) {
	q := New(cmp.Compare[int], 2, 1)
	c := q.Clone()
	c.Pop()
	c.Push(0)
	if q.Len() != 2 || q.Peek().MustGet() != 1 || c.Peek().MustGet() != 0 {
		t.Errorf("Clone shares storage: q top %v, c top %v", q.Peek(), c.Peek())
	}
}

func TestClearAndAll(t *testing.T) {
	q := New(cmp.Compare[int], 2, 1, 3)
	if got := slices.Sorted(q.All()); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("All = %v", got)
	}
	if q.Len() != 3 {
		t.Errorf("All consumed elements: len %d", q.Len())
	}
	q.Clear()
	if !q.IsEmpty() {
		t.Error("not empty after Clear")
	}
	q.Push(5)
	if got := q.Pop().MustGet(); got != 5 {
		t.Errorf("Pop after Clear = %d, want 5", got)
	}
}

func BenchmarkPushPop(b *testing.B) {
	q := New(cmp.Compare[int])
	rng := rand.New(rand.NewPCG(1, 2))
	for range 1024 {
		q.Push(rng.Int())
	}
	for b.Loop() {
		q.Push(rng.Int())
		q.Pop()
	}
}

func TestString(t *testing.T) {
	q := New(cmp.Compare[int], 3, 1, 2)
	if got := q.String(); got != "Queue[1 2 3]" {
		t.Errorf("String() = %q", got)
	}
	if q.Len() != 3 {
		t.Errorf("String() removed elements")
	}
	if got := New(cmp.Compare[int]).String(); got != "Queue[]" {
		t.Errorf("String() of an empty Queue = %q", got)
	}
}
