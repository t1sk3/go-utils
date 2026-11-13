package ring

import (
	"slices"
	"testing"
)

func TestNewPanics(t *testing.T) {
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d) did not panic", n)
				}
			}()
			New[int](n)
		}()
	}
}

func TestEmpty(t *testing.T) {
	r := New[int](3)
	if !r.IsEmpty() || r.IsFull() || r.Len() != 0 || r.Cap() != 3 {
		t.Errorf("a new Ring is not empty")
	}
	if r.Pop().IsPresent() || r.Oldest().IsPresent() || r.Newest().IsPresent() {
		t.Errorf("an empty Ring returned a value")
	}
	if got := r.ToSlice(); got == nil || len(got) != 0 {
		t.Errorf("ToSlice() of an empty Ring = %#v", got)
	}
	if r.String() != "Ring[]" {
		t.Errorf("String() = %q", r)
	}
}

func TestOverwritesOldest(t *testing.T) {
	r := New[int](3)
	for i := 1; i <= 3; i++ {
		if r.Push(i).IsPresent() {
			t.Errorf("Push(%d) evicted a value before the Ring was full", i)
		}
	}
	if !r.IsFull() || r.String() != "Ring[1 2 3]" {
		t.Errorf("after three pushes: %v", r)
	}
	for i := 4; i <= 8; i++ {
		if got, ok := r.Push(i).Value(); !ok || got != i-3 {
			t.Errorf("Push(%d) evicted %v, %v, want %d", i, got, ok, i-3)
		}
	}
	if got := r.ToSlice(); !slices.Equal(got, []int{6, 7, 8}) {
		t.Errorf("ToSlice() = %v", got)
	}
	if got := slices.Collect(r.All()); !slices.Equal(got, []int{6, 7, 8}) {
		t.Errorf("All() = %v", got)
	}
	if got := slices.Collect(r.Backward()); !slices.Equal(got, []int{8, 7, 6}) {
		t.Errorf("Backward() = %v", got)
	}
	if r.Oldest().OrElse(0) != 6 || r.Newest().OrElse(0) != 8 || r.At(1) != 7 || r.Len() != 3 {
		t.Errorf("Oldest, Newest or At is wrong: %v", r)
	}
	for range r.All() {
		break
	}
	for range r.Backward() {
		break
	}
}

func TestPopAndRefill(t *testing.T) {
	r := New[*int](2)
	one, two, three := new(1), new(2), new(3)
	r.Push(one)
	r.Push(two)
	r.Push(three)
	if got := r.Pop().OrElse(nil); got != two {
		t.Errorf("Pop() did not return the oldest value")
	}
	if r.buf[0] != three || r.buf[1] != nil {
		t.Errorf("Pop() left its slot holding a pointer")
	}
	if r.IsFull() || r.Len() != 1 {
		t.Errorf("Len() = %d after a Pop", r.Len())
	}
	r.Push(one)
	if got := r.ToSlice(); !slices.Equal(got, []*int{three, one}) {
		t.Errorf("values after refilling are in the wrong order")
	}
	r.Clear()
	if !r.IsEmpty() || r.buf[0] != nil || r.buf[1] != nil || r.Cap() != 2 {
		t.Errorf("Clear left values behind")
	}
	r.Push(two)
	if r.Oldest().OrElse(nil) != two {
		t.Errorf("Push after Clear did not start over")
	}
}

func TestAtOutOfRange(t *testing.T) {
	r := New[int](2)
	r.Push(1)
	for _, i := range []int{-1, 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("At(%d) did not panic", i)
				}
			}()
			r.At(i)
		}()
	}
}
