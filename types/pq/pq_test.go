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

func TestNewMinMaxCollect(t *testing.T) {
	if got := slices.Collect(NewMin(3, 1, 2).Drain()); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("NewMin drains as %v", got)
	}
	if got := slices.Collect(NewMax("b", "c", "a").Drain()); !slices.Equal(got, []string{"c", "b", "a"}) {
		t.Errorf("NewMax drains as %v", got)
	}
	src := []int{5, 1, 4}
	q := Collect(cmp.Compare[int], slices.Values(src))
	if got := slices.Collect(q.Drain()); !slices.Equal(got, []int{1, 4, 5}) || !slices.Equal(src, []int{5, 1, 4}) {
		t.Errorf("Collect drains as %v, source now %v", got, src)
	}
}

func TestPushPop(t *testing.T) {
	q := NewMin[int]()
	if got := q.PushPop(5); got != 5 || q.Len() != 0 {
		t.Errorf("PushPop on an empty Queue = %d, leaving %d", got, q.Len())
	}
	q.Push(3, 7, 9)
	if got := q.PushPop(1); got != 1 || q.Len() != 3 {
		t.Errorf("PushPop of the smallest value = %d", got)
	}
	if got := q.PushPop(3); got != 3 || q.Len() != 3 {
		t.Errorf("PushPop of a value equal to the first = %d", got)
	}
	if got := q.PushPop(8); got != 3 {
		t.Errorf("PushPop(8) = %d, want 3", got)
	}
	if got := slices.Collect(q.Drain()); !slices.Equal(got, []int{7, 8, 9}) {
		t.Errorf("after PushPop the Queue drains as %v", got)
	}

	// The three largest of a sequence, kept in a min-heap of three.
	top := NewMin[int]()
	for _, v := range []int{4, 9, 1, 7, 3, 8, 2} {
		if top.Len() < 3 {
			top.Push(v)
		} else {
			top.PushPop(v)
		}
	}
	if got := slices.Collect(top.Drain()); !slices.Equal(got, []int{7, 8, 9}) {
		t.Errorf("top three = %v", got)
	}
}

func TestRemoveFunc(t *testing.T) {
	q := NewMin(5, 2, 8, 1, 9, 4, 7, 3, 6)
	if got := q.RemoveFunc(func(v int) bool { return v%2 == 0 }); got != 4 {
		t.Errorf("RemoveFunc removed %d", got)
	}
	if got := q.RemoveFunc(func(v int) bool { return v > 100 }); got != 0 || q.Len() != 5 {
		t.Errorf("RemoveFunc with no match removed %d", got)
	}
	if got := slices.Collect(q.Drain()); !slices.Equal(got, []int{1, 3, 5, 7, 9}) {
		t.Errorf("after RemoveFunc the Queue drains as %v", got)
	}
}
