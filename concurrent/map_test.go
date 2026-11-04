package concurrent

import (
	"maps"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/t1sk3/go-utils/types/optional"
)

func TestZeroValue(t *testing.T) {
	var m Map[string, int]
	if m.Len() != 0 || m.Has("a") || m.Get("a").IsPresent() || m.Remove("a").IsPresent() {
		t.Error("the zero value reports a key")
	}
	m.Clear()
	if got := m.Snapshot(); got == nil || len(got) != 0 {
		t.Errorf("Snapshot = %#v", got)
	}
	for range m.All() {
		t.Error("All yielded a value")
	}
	if got := m.Put("a", 1); got.IsPresent() {
		t.Errorf("first Put = %v", got)
	}
}

func TestPutGetRemove(t *testing.T) {
	var m Map[string, int]
	m.Put("a", 1)
	if got := m.Put("a", 2); got != optional.Of(1) {
		t.Errorf("Put returned %v", got)
	}
	if got := m.Get("a"); got != optional.Of(2) || !m.Has("a") || m.Len() != 1 {
		t.Errorf("Get = %v, Len = %d", got, m.Len())
	}
	if got := m.Remove("a"); got != optional.Of(2) || m.Has("a") {
		t.Errorf("Remove = %v", got)
	}
	m.Put("b", 1)
	m.Clear()
	if m.Len() != 0 {
		t.Errorf("Len after Clear = %d", m.Len())
	}
}

func TestPutIfAbsent(t *testing.T) {
	var m Map[string, int]
	if v, stored := m.PutIfAbsent("a", 1); v != 1 || !stored {
		t.Errorf("first = %d, %v", v, stored)
	}
	if v, stored := m.PutIfAbsent("a", 2); v != 1 || stored {
		t.Errorf("second = %d, %v", v, stored)
	}
}

func TestCompute(t *testing.T) {
	var m Map[string, int]
	inc := func(v int, present bool) (int, bool) { return v + 1, true }
	if got := m.Compute("a", inc); got != optional.Of(1) {
		t.Errorf("Compute on a missing key = %v", got)
	}
	if got := m.Compute("a", inc); got != optional.Of(2) {
		t.Errorf("Compute on a present key = %v", got)
	}
	got := m.Compute("a", func(v int, present bool) (int, bool) {
		if !present || v != 2 {
			t.Errorf("f received %d, %v", v, present)
		}
		return 0, false
	})
	if got.IsPresent() || m.Has("a") {
		t.Errorf("after removing: %v", got)
	}
	if got := m.Compute("missing", func(int, bool) (int, bool) { return 0, false }); got.IsPresent() || m.Len() != 0 {
		t.Errorf("removing a missing key: %v", got)
	}
}

func TestMerge(t *testing.T) {
	var m Map[string, int]
	add := func(a, b int) int { return a + b }
	if got := m.Merge("a", 1, add); got != 1 {
		t.Errorf("first Merge = %d", got)
	}
	if got := m.Merge("a", 2, add); got != 3 {
		t.Errorf("second Merge = %d", got)
	}
}

func TestComputeIfAbsentRunsOncePerKey(t *testing.T) {
	var m Map[int, int]
	var calls atomic.Int64
	var wg sync.WaitGroup
	const goroutines, keys = 32, 10
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range keys {
				got := m.ComputeIfAbsent(k, func(k int) int {
					calls.Add(1)
					return k * 2
				})
				if got != k*2 {
					t.Errorf("ComputeIfAbsent(%d) = %d", k, got)
				}
			}
		}()
	}
	wg.Wait()
	if got := calls.Load(); got != keys {
		t.Errorf("f ran %d times for %d keys", got, keys)
	}
}

// TestConcurrentUpdates loses counts if compound updates are not atomic, and the race
// detector fails it if any path is unlocked.
func TestConcurrentUpdates(t *testing.T) {
	var m Map[string, int]
	var wg sync.WaitGroup
	const goroutines, rounds = 16, 500
	add := func(a, b int) int { return a + b }
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				m.Merge("merge", 1, add)
				m.Compute("compute", func(v int, _ bool) (int, bool) { return v + 1, true })
				m.Get("merge")
				m.Has("compute")
				m.Len()
				for range m.All() {
					break
				}
			}
		}()
	}
	wg.Wait()
	want := map[string]int{"merge": goroutines * rounds, "compute": goroutines * rounds}
	if got := m.Snapshot(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSnapshotIsIndependent(t *testing.T) {
	var m Map[string, int]
	m.Put("a", 1)
	s := m.Snapshot()
	s["b"] = 2
	m.Put("c", 3)
	if m.Has("b") || len(s) != 2 {
		t.Errorf("Snapshot shares storage: map %v, snapshot %v", m.Snapshot(), s)
	}
}

func TestAllAllowsCallsDuringLoop(t *testing.T) {
	var m Map[int, int]
	for i := range 10 {
		m.Put(i, i)
	}
	count := 0
	for k := range m.All() { // This would deadlock if the loop held the lock.
		m.Remove(k)
		count++
	}
	if count != 10 || m.Len() != 0 {
		t.Errorf("visited %d, left %d", count, m.Len())
	}
}

func BenchmarkGetParallel(b *testing.B) {
	var m Map[int, int]
	for i := range 1024 {
		m.Put(i, i)
	}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Get(i & 1023)
			i++
		}
	})
}

func BenchmarkMergeParallel(b *testing.B) {
	var m Map[int, int]
	add := func(a, c int) int { return a + c }
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Merge(i&1023, 1, add)
			i++
		}
	})
}
