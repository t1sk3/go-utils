package deque

import (
	"slices"
	"testing"
)

func TestBufferShrinksAfterBurst(t *testing.T) {
	for name, pop := range map[string]func(*Deque[int]){
		"PopFront": func(d *Deque[int]) { d.PopFront() },
		"PopBack":  func(d *Deque[int]) { d.PopBack() },
	} {
		t.Run(name, func(t *testing.T) {
			var d Deque[int]
			for i := range 10_000 {
				d.PushBack(i)
			}
			peak := len(d.buf)
			for d.Len() > 3 {
				pop(&d)
			}
			if len(d.buf) != minCapacity {
				t.Errorf("buffer holds %d slots for %d values, down from %d", len(d.buf), d.Len(), peak)
			}
			want := []int{0, 1, 2}
			if name == "PopFront" {
				want = []int{9997, 9998, 9999}
			}
			if got := slices.Collect(d.All()); !slices.Equal(got, want) {
				t.Errorf("values after shrinking = %v, want %v", got, want)
			}
		})
	}
}

// Pushing and popping around a resize boundary must not resize on every call.
func TestNoResizeThrashing(t *testing.T) {
	var d Deque[int]
	for i := range 65 {
		d.PushBack(i)
	}
	resizes, last := 0, len(d.buf)
	for range 1000 {
		d.PopBack()
		d.PushBack(0)
		if len(d.buf) != last {
			resizes, last = resizes+1, len(d.buf)
		}
	}
	if resizes != 0 {
		t.Errorf("%d resizes while the length moved between 64 and 65", resizes)
	}
}
