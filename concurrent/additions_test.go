package concurrent

import (
	"maps"
	"slices"
	"sync"
	"testing"
)

func TestMapAdditions(t *testing.T) {
	var m Map[string, int]
	if m.GetOrDefault("a", 7) != 7 || m.Has("a") {
		t.Errorf("GetOrDefault of a missing key did not return the fallback, or stored it")
	}
	if m.Replace("a", 1).IsPresent() || m.Has("a") {
		t.Errorf("Replace of a missing key stored a value")
	}
	called := false
	if m.ComputeIfPresent("a", func(int) (int, bool) { called = true; return 1, true }).IsPresent() || called || m.Has("a") {
		t.Errorf("ComputeIfPresent ran for a missing key")
	}
	m.Put("a", 0)
	if m.GetOrDefault("a", 7) != 0 {
		t.Errorf("GetOrDefault of a present zero value returned the fallback")
	}
	if old, ok := m.Replace("a", 5).Value(); !ok || old != 0 || m.Get("a").OrElse(0) != 5 {
		t.Errorf("Replace = %v, %v", old, ok)
	}
	if v, ok := m.ComputeIfPresent("a", func(v int) (int, bool) { return v * 2, true }).Value(); !ok || v != 10 {
		t.Errorf("ComputeIfPresent = %v, %v", v, ok)
	}
	if m.ComputeIfPresent("a", func(int) (int, bool) { return 0, false }).IsPresent() || m.Has("a") {
		t.Errorf("ComputeIfPresent returning false did not remove the entry")
	}

	for i, k := range []string{"a", "b", "c", "d"} {
		m.Put(k, i)
	}
	if got := slices.Sorted(m.Keys()); !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("Keys() = %v", got)
	}
	if got := slices.Sorted(m.Values()); !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Errorf("Values() = %v", got)
	}
	for range m.Keys() {
		break
	}
	for range m.Values() {
		break
	}
	if n := m.RemoveIf(func(_ string, v int) bool { return v%2 == 1 }); n != 2 || m.Len() != 2 || m.Has("b") {
		t.Errorf("RemoveIf removed %d, leaving %v", n, m.Snapshot())
	}
	var empty Map[string, int]
	if empty.RemoveIf(func(string, int) bool { return true }) != 0 {
		t.Errorf("RemoveIf on the zero value removed something")
	}
}

func TestCompareAndSwap(t *testing.T) {
	var m Map[string, int]
	if CompareAndSwap(&m, "a", 0, 1) || CompareAndDelete(&m, "a", 0) || m.Has("a") {
		t.Errorf("a compare operation succeeded on a missing key")
	}
	m.Put("a", 1)
	if CompareAndSwap(&m, "a", 2, 3) || !CompareAndSwap(&m, "a", 1, 3) || m.Get("a").OrElse(0) != 3 {
		t.Errorf("CompareAndSwap is wrong: %v", m.Snapshot())
	}
	if CompareAndDelete(&m, "a", 1) || !CompareAndDelete(&m, "a", 3) || m.Has("a") {
		t.Errorf("CompareAndDelete is wrong: %v", m.Snapshot())
	}

	// Many goroutines incrementing with CompareAndSwap lose no update.
	m.Put("n", 0)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 500 {
				for {
					v := m.Get("n").OrElse(0)
					if CompareAndSwap(&m, "n", v, v+1) {
						break
					}
				}
			}
		})
	}
	wg.Wait()
	if got := m.Get("n").OrElse(0); got != 4000 {
		t.Errorf("counter = %d after 4000 increments", got)
	}
}

func TestShardedMap(t *testing.T) {
	for _, shards := range []int{0, 1, 3, 16} {
		m := NewShardedMap[int, int](shards)
		if n := len(m.shards); n&(n-1) != 0 || n < max(shards, 1) {
			t.Errorf("NewShardedMap(%d) has %d shards, want a power of two at least that", shards, n)
		}
		want := map[int]int{}
		for i := range 200 {
			if m.Put(i, i).IsPresent() {
				t.Fatalf("Put of new key %d returned a previous value", i)
			}
			want[i] = i
		}
		if m.Len() != 200 || !maps.Equal(m.Snapshot(), want) {
			t.Errorf("%d shards: Len() = %d", shards, m.Len())
		}
		if v, ok := m.Get(7).Value(); !ok || v != 7 || !m.Has(7) || m.Has(999) || m.GetOrDefault(999, -1) != -1 {
			t.Errorf("lookups are wrong")
		}
		if old, ok := m.Put(7, 70).Value(); !ok || old != 7 {
			t.Errorf("Put over a key = %v, %v", old, ok)
		}
		if v, stored := m.PutIfAbsent(7, 0); stored || v != 70 {
			t.Errorf("PutIfAbsent on a present key = %v, %v", v, stored)
		}
		if old, ok := m.Replace(7, 71).Value(); !ok || old != 70 || m.Replace(999, 1).IsPresent() {
			t.Errorf("Replace is wrong")
		}
		if m.ComputeIfAbsent(1000, func(k int) int { return k * 2 }) != 2000 {
			t.Errorf("ComputeIfAbsent is wrong")
		}
		if v := m.ComputeIfPresent(1000, func(v int) (int, bool) { return v + 1, true }).OrElse(0); v != 2001 {
			t.Errorf("ComputeIfPresent = %d", v)
		}
		if v := m.Compute(1000, func(v int, ok bool) (int, bool) { return v + 1, ok }).OrElse(0); v != 2002 {
			t.Errorf("Compute = %d", v)
		}
		if m.Merge(1000, 8, func(a, b int) int { return a + b }) != 2010 {
			t.Errorf("Merge is wrong")
		}
		if v, ok := m.Remove(1000).Value(); !ok || v != 2010 || m.Has(1000) {
			t.Errorf("Remove = %v, %v", v, ok)
		}
		if n := m.RemoveIf(func(k, _ int) bool { return k >= 100 }); n != 100 || m.Len() != 100 {
			t.Errorf("RemoveIf removed %d, leaving %d", n, m.Len())
		}
		for range m.All() {
			break
		}
		m.Clear()
		if m.Len() != 0 || len(m.Snapshot()) != 0 {
			t.Errorf("Clear left %d entries", m.Len())
		}
	}
}

func TestShardedMapConcurrent(t *testing.T) {
	m := NewShardedMap[int, int](8)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 500 {
				m.Merge(i%50, 1, func(a, b int) int { return a + b })
				m.Get(i)
				m.Len()
				if g == 0 && i%100 == 0 {
					for range m.All() {
					}
				}
			}
		})
	}
	wg.Wait()
	total := 0
	for _, v := range m.All() {
		total += v
	}
	if total != 4000 || m.Len() != 50 {
		t.Errorf("total = %d over %d keys, want 4000 over 50", total, m.Len())
	}
}

func TestSet(t *testing.T) {
	var s Set[string]
	if s.Has("a") || s.Remove("a") || s.Len() != 0 {
		t.Errorf("the zero value is not empty")
	}
	if got := s.Snapshot(); got == nil || len(got) != 0 {
		t.Errorf("Snapshot() of the zero value = %#v", got)
	}
	s.Clear()
	if !s.Add("a") || s.Add("a") || !s.Add("b") || s.Len() != 2 || !s.Has("a") {
		t.Errorf("Add is wrong: %v", s.Snapshot())
	}
	if got := slices.Sorted(s.All()); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("All() = %v", got)
	}
	for range s.All() {
		break
	}
	if !s.Remove("a") || s.Remove("a") || s.Has("a") {
		t.Errorf("Remove is wrong")
	}
	s.Clear()
	if s.Len() != 0 {
		t.Errorf("Clear left %d values", s.Len())
	}

	// Add lets exactly one goroutine claim each value.
	var claims Map[int, int]
	var wg sync.WaitGroup
	var ints Set[int]
	for range 8 {
		wg.Go(func() {
			for i := range 200 {
				if ints.Add(i) {
					claims.Merge(i, 1, func(a, b int) int { return a + b })
				}
			}
		})
	}
	wg.Wait()
	if claims.Len() != 200 || claims.RemoveIf(func(_, n int) bool { return n == 1 }) != 200 {
		t.Errorf("some value was claimed by more than one goroutine, or by none")
	}
}
