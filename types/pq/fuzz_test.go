package pq

import (
	"cmp"
	"slices"
	"testing"
)

// FuzzQueue runs a sequence of pushes and pops against a Queue and against a sorted
// slice, and checks that every Pop returns the smallest element.
func FuzzQueue(f *testing.F) {
	f.Add([]byte{5, 3, 8, 0, 1, 0, 0, 9, 2})
	f.Fuzz(func(t *testing.T, ops []byte) {
		half := len(ops) / 2
		initial := make([]int, half)
		for i, b := range ops[:half] {
			initial[i] = int(b)
		}
		q := New(cmp.Compare[int], initial...)
		model := slices.Sorted(slices.Values(initial))
		for _, b := range ops[half:] {
			if b%3 == 0 {
				got, ok := q.Pop().Value()
				if ok != (len(model) > 0) || (ok && got != model[0]) {
					t.Fatalf("Pop() = %v, %v with %v", got, ok, model)
				}
				if ok {
					model = model[1:]
				}
			} else {
				q.Push(int(b))
				i, _ := slices.BinarySearch(model, int(b))
				model = slices.Insert(model, i, int(b))
			}
			if peek, ok := q.Peek().Value(); ok != (len(model) > 0) || (ok && peek != model[0]) {
				t.Fatalf("Peek() = %v, %v with %v", peek, ok, model)
			}
		}
		if got := slices.Collect(q.Drain()); !slices.Equal(got, model) {
			t.Fatalf("Drain() = %v, want %v", got, model)
		}
	})
}
