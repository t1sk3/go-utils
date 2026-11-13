package deque

import (
	"encoding/json"
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

func TestSet(t *testing.T) {
	d := Of(1, 2, 3)
	d.PopFront()
	d.PushBack(4) // 2 3 4, not starting at the buffer's first slot
	d.Set(0, 20)
	d.Set(2, 40)
	if got := slices.Collect(d.All()); !slices.Equal(got, []int{20, 3, 40}) {
		t.Errorf("after Set: %v", got)
	}
	for _, i := range []int{-1, 3} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Set(%d) did not panic", i)
				}
			}()
			d.Set(i, 0)
		}()
	}
}

func TestRotate(t *testing.T) {
	// Each layout puts the same values 0..n-1 in the Deque with the buffer in a
	// different state: starting at slot 0, wrapped around the end, and completely full.
	layouts := map[string]func(n int) *Deque[int]{
		"plain": func(n int) *Deque[int] {
			d := &Deque[int]{}
			for i := range n {
				d.PushBack(i)
			}
			return d
		},
		"wrapped": func(n int) *Deque[int] {
			d := &Deque[int]{}
			for i := n - 1; i >= 0; i-- {
				d.PushFront(i)
			}
			return d
		},
	}
	for name, build := range layouts {
		for _, n := range []int{0, 1, 2, 5, 8, 9} {
			for by := -2*n - 1; by <= 2*n+1; by++ {
				d := build(n)
				d.Rotate(by)
				want := make([]int, n)
				for i := range n {
					want[i] = ((i+by)%n + n) % n
				}
				if got := slices.Collect(d.All()); !slices.Equal(got, want) {
					t.Fatalf("%s: Rotate(%d) of %d values = %v, want %v", name, by, n, got, want)
				}
				// Slots outside the values must not keep copies that pin memory.
				used := map[int]bool{}
				for i := range d.n {
					used[d.index(i)] = true
				}
				for i, v := range d.buf {
					if !used[i] && v != 0 {
						t.Fatalf("%s: Rotate(%d) of %d values left %d in unused slot %d", name, by, n, v, i)
					}
				}
			}
		}
	}
}

func TestJSON(t *testing.T) {
	d := Of(1, 2, 3)
	d.PopFront()
	d.PushBack(4)
	data, err := json.Marshal(d)
	if err != nil || string(data) != "[2,3,4]" {
		t.Fatalf("Marshal = %s, %v", data, err)
	}
	if data, _ := json.Marshal(&Deque[int]{}); string(data) != "[]" {
		t.Errorf("Marshal of an empty Deque = %s", data)
	}
	var back Deque[int]
	back.PushBack(9)
	if err := json.Unmarshal(data, &back); err != nil || !slices.Equal(slices.Collect(back.All()), []int{2, 3, 4}) {
		t.Errorf("Unmarshal = %v, %v", &back, err)
	}
	if err := json.Unmarshal([]byte(`{"a":1}`), &back); err == nil {
		t.Errorf("Unmarshal of an object did not fail")
	}
	var wrapped struct{ Queue Deque[string] }
	if err := json.Unmarshal([]byte(`{"Queue":["x","y"]}`), &wrapped); err != nil || wrapped.Queue.Back().OrElse("") != "y" {
		t.Errorf("Unmarshal into a struct field = %v, %v", &wrapped.Queue, err)
	}
}
