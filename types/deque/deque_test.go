package deque

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestZeroValue(t *testing.T) {
	var d Deque[int]
	if !d.IsEmpty() || d.Len() != 0 {
		t.Fatalf("zero value is not empty: %v", &d)
	}
	for name, got := range map[string]bool{
		"PopFront": d.PopFront().IsEmpty(),
		"PopBack":  d.PopBack().IsEmpty(),
		"Front":    d.Front().IsEmpty(),
		"Back":     d.Back().IsEmpty(),
	} {
		if !got {
			t.Errorf("%s on an empty Deque returned a value", name)
		}
	}
	d.PushFront(1)
	if got := d.Back().MustGet(); got != 1 {
		t.Errorf("Back after PushFront on the zero value = %d, want 1", got)
	}
}

func TestPushFrontOrder(t *testing.T) {
	d := Of(3)
	d.PushFront(2, 1)
	if got := slices.Collect(d.All()); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("got %v, want [1 2 3]", got)
	}
}

func TestQueueAndStack(t *testing.T) {
	queue := Of(1, 2, 3)
	if got := queue.PopFront().MustGet(); got != 1 {
		t.Errorf("PopFront = %d, want 1", got)
	}
	stack := Of(1, 2, 3)
	if got := stack.PopBack().MustGet(); got != 3 {
		t.Errorf("PopBack = %d, want 3", got)
	}
}

// TestAgainstSlice runs random operations against a slice with the same contents,
// which exercises growth while the values wrap around the end of the buffer.
func TestAgainstSlice(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	var d Deque[int]
	var model []int
	for i := range 20_000 {
		switch rng.IntN(6) {
		case 0, 1:
			d.PushBack(i)
			model = append(model, i)
		case 2:
			d.PushFront(i)
			model = slices.Insert(model, 0, i)
		case 3:
			got, ok := d.PopFront().Value()
			if ok != (len(model) > 0) {
				t.Fatalf("PopFront ok = %v with %d values", ok, len(model))
			}
			if ok {
				if got != model[0] {
					t.Fatalf("PopFront = %d, want %d", got, model[0])
				}
				model = model[1:]
			}
		case 4:
			got, ok := d.PopBack().Value()
			if ok != (len(model) > 0) {
				t.Fatalf("PopBack ok = %v with %d values", ok, len(model))
			}
			if ok {
				if got != model[len(model)-1] {
					t.Fatalf("PopBack = %d, want %d", got, model[len(model)-1])
				}
				model = model[:len(model)-1]
			}
		case 5:
			d.PushBack(i, i+1, i+2)
			model = append(model, i, i+1, i+2)
		}
		if d.Len() != len(model) {
			t.Fatalf("Len = %d, want %d", d.Len(), len(model))
		}
	}
	if got := slices.Collect(d.All()); !slices.Equal(got, model) {
		t.Errorf("All differs from the model")
	}
	backward := slices.Collect(d.Backward())
	slices.Reverse(backward)
	if !slices.Equal(backward, model) {
		t.Errorf("Backward differs from the model")
	}
	for i, want := range model {
		if got := d.At(i); got != want {
			t.Fatalf("At(%d) = %d, want %d", i, got, want)
		}
	}
}

func TestAtOutOfRange(t *testing.T) {
	d := Of(1)
	for _, i := range []int{-1, 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("At(%d) did not panic", i)
				}
			}()
			d.At(i)
		}()
	}
}

func TestPopClearsSlot(t *testing.T) {
	v := 1
	d := Of(&v, &v)
	d.PopFront()
	d.PopBack()
	for i, p := range d.buf {
		if p != nil {
			t.Errorf("slot %d still refers to a removed value", i)
		}
	}
}

func TestClear(t *testing.T) {
	d := Of(1, 2, 3)
	capacity := len(d.buf)
	d.Clear()
	if !d.IsEmpty() || len(d.buf) != capacity {
		t.Errorf("after Clear: len %d, capacity %d, want 0, %d", d.Len(), len(d.buf), capacity)
	}
	d.PushBack(4)
	if got := d.Front().MustGet(); got != 4 {
		t.Errorf("Front after Clear = %d, want 4", got)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	d := Of(1, 2)
	d.PopFront() // Leave the head away from the start of the buffer.
	d.PushBack(3)
	c := d.Clone()
	c.PushBack(4)
	d.PopBack()
	if got := fmt.Sprint(d, c); got != "Deque[2] Deque[2 3 4]" {
		t.Errorf("got %s", got)
	}
}

func TestIteratorsStopEarly(t *testing.T) {
	d := Of(1, 2, 3)
	for name, seq := range map[string]func(func(int) bool){"All": d.All(), "Backward": d.Backward()} {
		calls := 0
		for range seq {
			calls++
			break
		}
		if calls != 1 {
			t.Errorf("%s made %d calls after break", name, calls)
		}
	}
}

func TestCollect(t *testing.T) {
	d := Collect(slices.Values([]string{"a", "b"}))
	if got := d.String(); got != "Deque[a b]" {
		t.Errorf("got %s", got)
	}
}

func BenchmarkQueue(b *testing.B) {
	var d Deque[int]
	d.PushBack(make([]int, 1024)...)
	for b.Loop() {
		d.PushBack(1)
		d.PopFront()
	}
}
