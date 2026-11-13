package deque

import (
	"slices"
	"testing"
)

// FuzzDeque runs a sequence of operations against a Deque and against a slice, and
// checks that they agree and that the ring buffer's bookkeeping stays in range.
func FuzzDeque(f *testing.F) {
	f.Add([]byte{0, 1, 0, 2, 1, 3, 2, 0, 3, 0})
	f.Add([]byte{1, 1, 1, 2, 1, 3, 1, 4, 1, 5, 1, 6, 1, 7, 1, 8, 1, 9, 2, 0, 2, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		var d Deque[int]
		var model []int
		for i := 0; i+1 < len(ops); i += 2 {
			op, arg := ops[i]%6, int(ops[i+1])
			switch op {
			case 0:
				d.PushBack(arg)
				model = append(model, arg)
			case 1:
				d.PushFront(arg)
				model = slices.Insert(model, 0, arg)
			case 2:
				got, ok := d.PopFront().Value()
				if ok != (len(model) > 0) || (ok && got != model[0]) {
					t.Fatalf("PopFront() = %v, %v with %v", got, ok, model)
				}
				if ok {
					model = model[1:]
				}
			case 3:
				got, ok := d.PopBack().Value()
				if ok != (len(model) > 0) || (ok && got != model[len(model)-1]) {
					t.Fatalf("PopBack() = %v, %v with %v", got, ok, model)
				}
				if ok {
					model = model[:len(model)-1]
				}
			case 4:
				// Several values at once take the path that grows by more than one step.
				values := make([]int, arg%40)
				for j := range values {
					values[j] = arg + j
				}
				d.PushBack(values...)
				model = append(model, values...)
			case 5:
				if arg%16 == 0 {
					d.Clear()
					model = model[:0]
				}
			}
			if d.Len() != len(model) || d.n > len(d.buf) || d.head < 0 || (len(d.buf) > 0 && d.head >= len(d.buf)) {
				t.Fatalf("Len() = %d, head = %d, buffer of %d, want length %d", d.Len(), d.head, len(d.buf), len(model))
			}
		}
		if got := slices.Collect(d.All()); !slices.Equal(got, model) {
			t.Fatalf("All() = %v, want %v", got, model)
		}
		back := slices.Collect(d.Backward())
		slices.Reverse(back)
		if !slices.Equal(back, model) {
			t.Fatalf("Backward() reversed = %v, want %v", back, model)
		}
		for i, want := range model {
			if got := d.At(i); got != want {
				t.Fatalf("At(%d) = %d, want %d", i, got, want)
			}
		}
	})
}
