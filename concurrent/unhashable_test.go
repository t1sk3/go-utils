package concurrent

import (
	"testing"
	"time"
)

// A lookup with an unhashable key panics. Every method must release its lock on the way
// out, or the Map is unusable for whoever recovers.
func TestUnhashableKeyDoesNotLeakLock(t *testing.T) {
	bad := []int{1}
	calls := map[string]func(m *Map[any, int]){
		"Get":             func(m *Map[any, int]) { m.Get(bad) },
		"Has":             func(m *Map[any, int]) { m.Has(bad) },
		"Put":             func(m *Map[any, int]) { m.Put(bad, 1) },
		"PutIfAbsent":     func(m *Map[any, int]) { m.PutIfAbsent(bad, 1) },
		"Remove":          func(m *Map[any, int]) { m.Remove(bad) },
		"ComputeIfAbsent": func(m *Map[any, int]) { m.ComputeIfAbsent(bad, func(any) int { return 1 }) },
		"Compute":         func(m *Map[any, int]) { m.Compute(bad, func(int, bool) (int, bool) { return 1, true }) },
		"Merge":           func(m *Map[any, int]) { m.Merge(bad, 1, func(a, b int) int { return a + b }) },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var m Map[any, int]
			m.Put("a", 1)
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("no panic for an unhashable key")
					}
				}()
				call(&m)
			}()
			done := make(chan struct{})
			go func() {
				defer close(done)
				m.Put("b", 2)
				m.Get("a")
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("the Map is still locked after the panic")
			}
		})
	}
}
