package sorted

import (
	"slices"
	"testing"
)

// FuzzMap runs a sequence of operations against a Map and against a plain map with a
// sorted key slice, and checks that they agree and that the tree stays balanced.
func FuzzMap(f *testing.F) {
	f.Add([]byte{0, 5, 0, 3, 0, 8, 1, 5, 2, 4, 3, 0, 4, 0})
	f.Add([]byte{0, 1, 0, 2, 0, 3, 0, 4, 0, 5, 0, 6, 0, 7, 1, 4, 1, 2, 1, 6})
	f.Fuzz(func(t *testing.T, ops []byte) {
		m := NewOrderedMap[int, int]()
		model := map[int]int{}
		for i := 0; i+1 < len(ops); i += 2 {
			op, key := ops[i]%6, int(ops[i+1]%64)
			switch op {
			case 0, 1:
				old, had := model[key]
				got, ok := m.Put(key, i).Value()
				if ok != had || got != old {
					t.Fatalf("Put(%d) returned %v, %v, want %v, %v", key, got, ok, old, had)
				}
				model[key] = i
			case 2:
				old, had := model[key]
				got, ok := m.Remove(key).Value()
				if ok != had || got != old {
					t.Fatalf("Remove(%d) returned %v, %v, want %v, %v", key, got, ok, old, had)
				}
				delete(model, key)
			case 3:
				if e, ok := m.PopFirst().Value(); ok {
					delete(model, e.First)
				}
			case 4:
				if e, ok := m.PopLast().Value(); ok {
					delete(model, e.First)
				}
			case 5:
				got, ok := m.Get(key).Value()
				if want, had := model[key]; ok != had || got != want {
					t.Fatalf("Get(%d) = %v, %v, want %v, %v", key, got, ok, want, had)
				}
			}
			checkInvariants(t, m)
		}

		keys := make([]int, 0, len(model))
		for k := range model {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		if got := slices.Collect(m.Keys()); !slices.Equal(got, keys) {
			t.Fatalf("Keys() = %v, want %v", got, keys)
		}
		for probe := -1; probe <= 64; probe++ {
			i, found := slices.BinarySearch(keys, probe)
			check := func(name string, got bool, key int, wantIndex int) {
				want := wantIndex >= 0 && wantIndex < len(keys)
				if got != want || (want && key != keys[wantIndex]) {
					t.Fatalf("%s(%d) = %v, %v with keys %v", name, probe, key, got, keys)
				}
			}
			floor, ceiling, lower, higher := i-1, i, i-1, i
			if found {
				floor, higher = i, i+1
			}
			e, ok := m.Floor(probe).Value()
			check("Floor", ok, e.First, floor)
			e, ok = m.Ceiling(probe).Value()
			check("Ceiling", ok, e.First, ceiling)
			e, ok = m.Lower(probe).Value()
			check("Lower", ok, e.First, lower)
			e, ok = m.Higher(probe).Value()
			check("Higher", ok, e.First, higher)
		}
	})
}
