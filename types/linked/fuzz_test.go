package linked

import (
	"slices"
	"testing"
)

// FuzzMap runs a sequence of operations against a Map and against a slice of keys in
// order, and checks that the hash index and the linked list stay in step.
func FuzzMap(f *testing.F) {
	f.Add([]byte{0, 1, 0, 2, 0, 3, 2, 1, 3, 3, 1, 2, 4, 0, 5, 0})
	f.Fuzz(func(t *testing.T, ops []byte) {
		m := NewMap[int, int]()
		var order []int
		remove := func(key int) {
			if i := slices.Index(order, key); i >= 0 {
				order = slices.Delete(order, i, i+1)
			}
		}
		for i := 0; i+1 < len(ops); i += 2 {
			op, key := ops[i]%6, int(ops[i+1]%16)
			present := slices.Contains(order, key)
			switch op {
			case 0:
				if had := m.Put(key, i).IsPresent(); had != present {
					t.Fatalf("Put(%d) reported a previous value: %v, want %v", key, had, present)
				}
				if !present {
					order = append(order, key)
				}
			case 1:
				if had := m.Remove(key).IsPresent(); had != present {
					t.Fatalf("Remove(%d) = %v, want %v", key, had, present)
				}
				remove(key)
			case 2:
				if m.MoveToBack(key) != present {
					t.Fatalf("MoveToBack(%d) != %v", key, present)
				}
				if present {
					remove(key)
					order = append(order, key)
				}
			case 3:
				if m.MoveToFront(key) != present {
					t.Fatalf("MoveToFront(%d) != %v", key, present)
				}
				if present {
					remove(key)
					order = slices.Insert(order, 0, key)
				}
			case 4:
				if e, ok := m.PopFirst().Value(); ok != (len(order) > 0) || (ok && e.First != order[0]) {
					t.Fatalf("PopFirst() = %v, %v with %v", e, ok, order)
				} else if ok {
					order = order[1:]
				}
			case 5:
				if e, ok := m.PopLast().Value(); ok != (len(order) > 0) || (ok && e.First != order[len(order)-1]) {
					t.Fatalf("PopLast() = %v, %v with %v", e, ok, order)
				} else if ok {
					order = order[:len(order)-1]
				}
			}
			if m.Len() != len(order) {
				t.Fatalf("Len() = %d, want %d", m.Len(), len(order))
			}
		}
		if got := slices.Collect(m.Keys()); !slices.Equal(got, order) {
			t.Fatalf("Keys() = %v, want %v", got, order)
		}
		var back []int
		for k := range m.Backward() {
			back = append(back, k)
		}
		slices.Reverse(back)
		if !slices.Equal(back, order) {
			t.Fatalf("Backward() reversed = %v, want %v", back, order)
		}
	})
}
