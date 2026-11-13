package linked

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// checkLinks verifies that the list matches the hash map in both directions.
func checkLinks[K comparable, V any](t *testing.T, m *Map[K, V]) {
	t.Helper()
	count := 0
	var prev *node[K, V]
	for n := m.first; n != nil; n = n.next {
		if n.prev != prev {
			t.Fatalf("node %v has the wrong prev", n.key)
		}
		if m.nodes[n.key] != n {
			t.Fatalf("node %v is not the one in the hash map", n.key)
		}
		prev = n
		count++
	}
	if prev != m.last {
		t.Fatal("last does not point at the end of the list")
	}
	if count != len(m.nodes) {
		t.Fatalf("list holds %d nodes, hash map holds %d", count, len(m.nodes))
	}
}

func TestZeroValue(t *testing.T) {
	var m Map[string, int]
	if !m.IsEmpty() || m.Len() != 0 || m.Has("a") || m.Get("a").IsPresent() {
		t.Error("the zero value reports a key")
	}
	if m.First().IsPresent() || m.Last().IsPresent() || m.PopFirst().IsPresent() || m.PopLast().IsPresent() {
		t.Error("the zero value returned an entry")
	}
	if m.Remove("a").IsPresent() || m.MoveToBack("a") || m.MoveToFront("a") {
		t.Error("the zero value acted on a missing key")
	}
	m.Clear()
	m.Put("a", 1)
	if got := m.String(); got != "Map[a:1]" {
		t.Errorf("String = %q", got)
	}
	var s Set[int]
	s.Add(1)
	if got := s.String(); got != "Set[1]" {
		t.Errorf("String = %q", got)
	}
}

func TestMapOrder(t *testing.T) {
	m := NewMap[string, int]()
	m.Put("c", 3)
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 30) // An existing key keeps its position.
	if got := m.String(); got != "Map[c:30 a:1 b:2]" {
		t.Errorf("String = %q", got)
	}
	if got := slices.Collect(m.Keys()); !slices.Equal(got, []string{"c", "a", "b"}) {
		t.Errorf("Keys = %v", got)
	}
	if got := slices.Collect(m.Values()); !slices.Equal(got, []int{30, 1, 2}) {
		t.Errorf("Values = %v", got)
	}
	var backward []string
	for k := range m.Backward() {
		backward = append(backward, k)
	}
	if !slices.Equal(backward, []string{"b", "a", "c"}) {
		t.Errorf("Backward = %v", backward)
	}
	if got := m.First().MustGet().First; got != "c" {
		t.Errorf("First = %s", got)
	}
	if got := m.Last().MustGet().First; got != "b" {
		t.Errorf("Last = %s", got)
	}
}

func TestMove(t *testing.T) {
	m := NewMap[int, int]()
	for i := 1; i <= 3; i++ {
		m.Put(i, i)
	}
	steps := []struct {
		name string
		do   func() bool
		want []int
	}{
		{"front to back", func() bool { return m.MoveToBack(1) }, []int{2, 3, 1}},
		{"back to back", func() bool { return m.MoveToBack(1) }, []int{2, 3, 1}},
		{"middle to front", func() bool { return m.MoveToFront(3) }, []int{3, 2, 1}},
		{"front to front", func() bool { return m.MoveToFront(3) }, []int{3, 2, 1}},
		{"back to front", func() bool { return m.MoveToFront(1) }, []int{1, 3, 2}},
		{"middle to back", func() bool { return m.MoveToBack(3) }, []int{1, 2, 3}},
	}
	for _, s := range steps {
		if !s.do() {
			t.Errorf("%s: reported the key as missing", s.name)
		}
		checkLinks(t, m)
		if got := slices.Collect(m.Keys()); !slices.Equal(got, s.want) {
			t.Errorf("%s: got %v, want %v", s.name, got, s.want)
		}
	}
	single := NewMap[int, int]()
	single.Put(1, 1)
	if !single.MoveToBack(1) || !single.MoveToFront(1) {
		t.Error("moving the only key failed")
	}
	checkLinks(t, single)
}

func TestAgainstModel(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := NewMap[int, int]()
	var order []int
	values := map[int]int{}
	for i := range 20_000 {
		k := rng.IntN(100)
		pos := slices.Index(order, k)
		switch rng.IntN(5) {
		case 0, 1:
			m.Put(k, i)
			values[k] = i
			if pos < 0 {
				order = append(order, k)
			}
		case 2:
			got, ok := m.Remove(k).Value()
			if ok != (pos >= 0) || (ok && got != values[k]) {
				t.Fatalf("Remove(%d) = %d, %v", k, got, ok)
			}
			if ok {
				order = slices.Delete(order, pos, pos+1)
			}
		case 3:
			if m.MoveToBack(k) != (pos >= 0) {
				t.Fatalf("MoveToBack(%d) disagrees with the model", k)
			}
			if pos >= 0 {
				order = append(slices.Delete(order, pos, pos+1), k)
			}
		case 4:
			if m.MoveToFront(k) != (pos >= 0) {
				t.Fatalf("MoveToFront(%d) disagrees with the model", k)
			}
			if pos >= 0 {
				order = slices.Insert(slices.Delete(order, pos, pos+1), 0, k)
			}
		}
		if i%100 == 0 {
			checkLinks(t, m)
			if got := slices.Collect(m.Keys()); !slices.Equal(got, order) {
				t.Fatalf("Keys = %v, want %v", got, order)
			}
		}
	}
}

func TestPop(t *testing.T) {
	m := NewMap[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)
	if got := m.PopFirst().MustGet(); got.First != "a" || got.Second != 1 {
		t.Errorf("PopFirst = %v", got)
	}
	if got := m.PopLast().MustGet(); got.First != "c" || got.Second != 3 {
		t.Errorf("PopLast = %v", got)
	}
	checkLinks(t, m)
	if got := m.String(); got != "Map[b:2]" {
		t.Errorf("String = %q", got)
	}
}

func TestRemoveCurrentDuringIteration(t *testing.T) {
	m := NewMap[int, int]()
	for i := range 6 {
		m.Put(i, i)
	}
	for k := range m.All() {
		if k%2 == 0 {
			m.Remove(k)
		}
	}
	if got := slices.Collect(m.Keys()); !slices.Equal(got, []int{1, 3, 5}) {
		t.Errorf("after All: %v", got)
	}
	for k := range m.Backward() {
		if k == 3 {
			m.Remove(k)
		}
	}
	checkLinks(t, m)
	if got := slices.Collect(m.Keys()); !slices.Equal(got, []int{1, 5}) {
		t.Errorf("after Backward: %v", got)
	}
}

func TestIteratorsStopEarly(t *testing.T) {
	m := NewMap[int, int]()
	m.Put(1, 1)
	m.Put(2, 2)
	calls := 0
	for range m.All() {
		calls++
		break
	}
	for range m.Backward() {
		calls++
		break
	}
	for range m.Keys() {
		calls++
		break
	}
	for range m.Values() {
		calls++
		break
	}
	s := NewSet(1, 2)
	for range s.All() {
		calls++
		break
	}
	for range s.Backward() {
		calls++
		break
	}
	if calls != 6 {
		t.Errorf("got %d calls, want 6", calls)
	}
}

func TestClear(t *testing.T) {
	m := NewMap[int, int]()
	m.Put(1, 1)
	m.Clear()
	if !m.IsEmpty() || m.First().IsPresent() || m.Last().IsPresent() {
		t.Error("not empty after Clear")
	}
	m.Put(2, 2)
	checkLinks(t, m)
}

func TestClone(t *testing.T) {
	m := NewMap[string, int]()
	m.Put("b", 2)
	m.Put("a", 1)
	c := m.Clone()
	c.Put("c", 3)
	m.Put("b", 9)
	if got := fmt.Sprint(m, c); got != "Map[b:9 a:1] Map[b:2 a:1 c:3]" {
		t.Errorf("got %s", got)
	}
	checkLinks(t, c)
}

func TestSet(t *testing.T) {
	s := NewSet("b", "a", "b", "c")
	if got := s.String(); got != "Set[b a c]" {
		t.Errorf("String = %q", got)
	}
	if s.Len() != 3 || !s.Has("a") || s.Has("z") || s.IsEmpty() {
		t.Errorf("got %v", s)
	}
	if got := slices.Collect(s.Backward()); !slices.Equal(got, []string{"c", "a", "b"}) {
		t.Errorf("Backward = %v", got)
	}
	if s.First().MustGet() != "b" || s.Last().MustGet() != "c" {
		t.Errorf("First = %v, Last = %v", s.First(), s.Last())
	}
	c := s.Clone()
	s.Remove("a", "missing")
	if got := fmt.Sprint(s, c); got != "Set[b c] Set[b a c]" {
		t.Errorf("got %s", got)
	}
	if !c.MoveToBack("b") || !c.MoveToFront("c") || c.MoveToBack("z") || c.MoveToFront("z") {
		t.Error("Move reported the wrong result")
	}
	if got := c.String(); got != "Set[c a b]" {
		t.Errorf("after moves: %q", got)
	}
	if c.PopFirst().MustGet() != "c" || c.PopLast().MustGet() != "b" {
		t.Error("Pop returned the wrong value")
	}
	if got := c.String(); got != "Set[a]" {
		t.Errorf("after pops: %q", got)
	}
	c.Clear()
	if !c.IsEmpty() || c.PopFirst().IsPresent() {
		t.Error("not empty after Clear")
	}
	if got := CollectSet(slices.Values([]int{2, 1, 2})).String(); got != "Set[2 1]" {
		t.Errorf("CollectSet = %q", got)
	}
}

func BenchmarkPutGet(b *testing.B) {
	m := NewMap[int, int]()
	for i := range 1 << 16 {
		m.Put(i, i)
	}
	i := 0
	for b.Loop() {
		k := i & (1<<16 - 1)
		m.Get(k)
		m.MoveToBack(k)
		i++
	}
}

func TestPutReturnsPrevious(t *testing.T) {
	m := NewMap[string, int]()
	if old := m.Put("a", 1); old.IsPresent() {
		t.Errorf("Put of a new key returned %v", old)
	}
	if old, ok := m.Put("a", 0).Value(); !ok || old != 1 {
		t.Errorf("Put of an existing key returned %v, %v", old, ok)
	}
	if old, ok := m.Put("a", 2).Value(); !ok || old != 0 {
		t.Errorf("Put over a zero value returned %v, %v", old, ok)
	}
}
