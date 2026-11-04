package sorted

import (
	"cmp"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// checkInvariants verifies the AVL balance and height of every node, the key order
// and the recorded size.
func checkInvariants[V any](t *testing.T, m *Map[int, V]) {
	t.Helper()
	count := 0
	var walk func(n *node[int, V], low, high *int) int8
	walk = func(n *node[int, V], low, high *int) int8 {
		if n == nil {
			return 0
		}
		count++
		if (low != nil && n.key <= *low) || (high != nil && n.key >= *high) {
			t.Fatalf("key %d is out of order", n.key)
		}
		l, r := walk(n.left, low, &n.key), walk(n.right, &n.key, high)
		if d := l - r; d < -1 || d > 1 {
			t.Fatalf("node %d is unbalanced: left %d, right %d", n.key, l, r)
		}
		if h := max(l, r) + 1; n.height != h {
			t.Fatalf("node %d has height %d, want %d", n.key, n.height, h)
		}
		return n.height
	}
	walk(m.root, nil, nil)
	if count != m.size {
		t.Fatalf("size is %d but the tree holds %d nodes", m.size, count)
	}
}

func optKey(e optional.Optional[pair.Pair[int, string]]) string {
	if p, ok := e.Value(); ok {
		return fmt.Sprint(p.First)
	}
	return "none"
}

func TestConstructorsRequireComparator(t *testing.T) {
	for name, f := range map[string]func(){
		"NewMap": func() { NewMap[int, int](nil) },
		"NewSet": func() { NewSet[int](nil) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s(nil) did not panic", name)
				}
			}()
			f()
		}()
	}
}

func TestAgainstModel(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := NewOrderedMap[int, int]()
	model := map[int]int{}
	for i := range 20_000 {
		k := rng.IntN(500)
		switch rng.IntN(3) {
		case 0, 1:
			m.Put(k, i)
			model[k] = i
		case 2:
			got, ok := m.Remove(k).Value()
			want, wantOK := model[k]
			if ok != wantOK || got != want {
				t.Fatalf("Remove(%d) = %d, %v, want %d, %v", k, got, ok, want, wantOK)
			}
			delete(model, k)
		}
		if i%500 == 0 {
			checkInvariants(t, m)
		}
		if got, want := m.Get(k), optional.OfOk(model[k], hasKey(model, k)); got != want {
			t.Fatalf("Get(%d) = %v, want %v", k, got, want)
		}
	}
	checkInvariants(t, m)
	if got := maps.Collect(m.All()); !maps.Equal(got, model) {
		t.Error("All differs from the model")
	}
	keys := slices.Sorted(maps.Keys(model))
	if got := slices.Collect(m.Keys()); !slices.Equal(got, keys) {
		t.Error("Keys are not sorted")
	}
}

func hasKey(m map[int]int, k int) bool {
	_, ok := m[k]
	return ok
}

// TestSequentialKeys covers the case that makes an unbalanced tree degrade to a list.
func TestSequentialKeys(t *testing.T) {
	m := NewOrderedMap[int, struct{}]()
	for i := range 1 << 14 {
		m.Put(i, struct{}{})
	}
	checkInvariants(t, m)
	if h := m.root.height; h > 20 {
		t.Errorf("height %d for %d sequential keys", h, m.Len())
	}
	for i := range 1 << 14 {
		if got := m.PopFirst().MustGet().First; got != i {
			t.Fatalf("PopFirst = %d, want %d", got, i)
		}
		if i%1000 == 0 {
			checkInvariants(t, m)
		}
	}
	if !m.IsEmpty() {
		t.Errorf("Len = %d after removing every key", m.Len())
	}
}

func TestNavigation(t *testing.T) {
	m := NewOrderedMap[int, string]()
	for _, k := range []int{20, 10, 30} {
		m.Put(k, fmt.Sprint(k))
	}
	tests := []struct {
		key                           int
		floor, ceiling, lower, higher string
	}{
		{5, "none", "10", "none", "10"},
		{10, "10", "10", "none", "20"},
		{15, "10", "20", "10", "20"},
		{20, "20", "20", "10", "30"},
		{30, "30", "30", "20", "none"},
		{35, "30", "none", "30", "none"},
	}
	for _, tt := range tests {
		got := []string{optKey(m.Floor(tt.key)), optKey(m.Ceiling(tt.key)), optKey(m.Lower(tt.key)), optKey(m.Higher(tt.key))}
		want := []string{tt.floor, tt.ceiling, tt.lower, tt.higher}
		if !slices.Equal(got, want) {
			t.Errorf("key %d: got %v, want %v", tt.key, got, want)
		}
	}
	if optKey(m.First()) != "10" || optKey(m.Last()) != "30" {
		t.Errorf("First = %v, Last = %v", m.First(), m.Last())
	}
}

func TestEmptyMap(t *testing.T) {
	m := NewOrderedMap[int, string]()
	for name, got := range map[string]optional.Optional[pair.Pair[int, string]]{
		"First": m.First(), "Last": m.Last(), "PopFirst": m.PopFirst(), "PopLast": m.PopLast(),
		"Floor": m.Floor(1), "Ceiling": m.Ceiling(1), "Lower": m.Lower(1), "Higher": m.Higher(1),
	} {
		if got.IsPresent() {
			t.Errorf("%s on an empty Map = %v", name, got)
		}
	}
	if m.Remove(1).IsPresent() || m.Has(1) || m.Get(1).IsPresent() {
		t.Error("an empty Map reports a key")
	}
	for range m.All() {
		t.Error("All yielded a value")
	}
	if got := m.String(); got != "Map[]" {
		t.Errorf("String = %q", got)
	}
}

func TestRange(t *testing.T) {
	m := NewOrderedMap[int, int]()
	for i := 0; i < 100; i += 10 {
		m.Put(i, i)
	}
	tests := []struct {
		from, to int
		want     []int
	}{
		{20, 50, []int{20, 30, 40}},
		{25, 50, []int{30, 40}},
		{-5, 15, []int{0, 10}},
		{85, 500, []int{90}},
		{50, 50, nil},
		{60, 20, nil},
		{1000, 2000, nil},
	}
	for _, tt := range tests {
		var got []int
		for k := range m.Range(tt.from, tt.to) {
			got = append(got, k)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("Range(%d, %d) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestIteratorsStopEarly(t *testing.T) {
	m := NewOrderedMap[int, int]()
	for i := range 50 {
		m.Put(i, i)
	}
	first := func(seq func(func(int, int) bool)) (int, int) {
		calls, key := 0, -1
		for k := range seq {
			calls++
			key = k
			break
		}
		return calls, key
	}
	if calls, k := first(m.All()); calls != 1 || k != 0 {
		t.Errorf("All: %d calls, key %d", calls, k)
	}
	if calls, k := first(m.Backward()); calls != 1 || k != 49 {
		t.Errorf("Backward: %d calls, key %d", calls, k)
	}
	if calls, k := first(m.Range(10, 20)); calls != 1 || k != 10 {
		t.Errorf("Range: %d calls, key %d", calls, k)
	}
	for range m.Keys() {
		break
	}
	for range m.Values() {
		break
	}
}

func TestPutReplaces(t *testing.T) {
	m := NewOrderedMap[string, int]()
	m.Put("a", 1)
	m.Put("a", 2)
	if m.Len() != 1 || m.Get("a").MustGet() != 2 {
		t.Errorf("got %v", m)
	}
}

func TestCustomComparatorAndString(t *testing.T) {
	m := NewMap[string, int](func(a, b string) int { return cmp.Compare(b, a) })
	m.Put("a", 1)
	m.Put("b", 2)
	if got := m.String(); got != "Map[b:2 a:1]" {
		t.Errorf("String = %q", got)
	}
	var values []int
	for v := range m.Values() {
		values = append(values, v)
	}
	if !slices.Equal(values, []int{2, 1}) {
		t.Errorf("Values = %v", values)
	}
}

func TestMapCloneIsIndependent(t *testing.T) {
	m := NewOrderedMap[string, int]()
	m.Put("a", 1)
	c := m.Clone()
	c.Put("b", 2)
	m.Put("a", 9)
	if got := fmt.Sprint(m, c); got != "Map[a:9] Map[a:1 b:2]" {
		t.Errorf("got %s", got)
	}
}

func TestMapClearAndPopLast(t *testing.T) {
	m := NewOrderedMap[int, int]()
	m.Put(1, 1)
	m.Put(2, 2)
	if got := m.PopLast().MustGet().First; got != 2 {
		t.Errorf("PopLast = %d", got)
	}
	m.Clear()
	if !m.IsEmpty() || m.First().IsPresent() {
		t.Error("not empty after Clear")
	}
}

func TestSet(t *testing.T) {
	s := NewOrderedSet(30, 10, 20, 10)
	if s.Len() != 3 || !s.Has(10) || s.Has(15) || s.IsEmpty() {
		t.Errorf("got %v", s)
	}
	if got := s.String(); got != "Set[10 20 30]" {
		t.Errorf("String = %q", got)
	}
	if got := slices.Collect(s.Backward()); !slices.Equal(got, []int{30, 20, 10}) {
		t.Errorf("Backward = %v", got)
	}
	if got := slices.Collect(s.Range(15, 30)); !slices.Equal(got, []int{20}) {
		t.Errorf("Range = %v", got)
	}
	got := []int{s.First().MustGet(), s.Last().MustGet(), s.Floor(15).MustGet(), s.Ceiling(15).MustGet(), s.Lower(10).OrElse(-1), s.Higher(30).OrElse(-1)}
	if want := []int{10, 30, 10, 20, -1, -1}; !slices.Equal(got, want) {
		t.Errorf("navigation = %v, want %v", got, want)
	}

	c := s.Clone()
	c.Remove(10, 99)
	if s.Len() != 3 || c.Len() != 2 {
		t.Errorf("Clone shares storage: %v, %v", s, c)
	}
	if got := c.PopFirst().MustGet(); got != 20 {
		t.Errorf("PopFirst = %d", got)
	}
	if got := c.PopLast().MustGet(); got != 30 {
		t.Errorf("PopLast = %d", got)
	}
	if c.PopFirst().IsPresent() {
		t.Error("PopFirst on an empty Set returned a value")
	}
	s.Clear()
	if !s.IsEmpty() {
		t.Error("not empty after Clear")
	}
}

func TestSetAddKeepsExistingEqualValue(t *testing.T) {
	type user struct {
		id   int
		name string
	}
	s := NewSet(func(a, b user) int { return cmp.Compare(a.id, b.id) }, user{1, "old"})
	s.Add(user{1, "new"})
	if got := s.First().MustGet().name; got != "old" || s.Len() != 1 {
		t.Errorf("got %q with %d values", got, s.Len())
	}
}

func BenchmarkPut(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := NewOrderedMap[int, int]()
	for b.Loop() {
		m.Put(rng.IntN(1<<20), 0)
	}
}

func BenchmarkGet(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := NewOrderedMap[int, int]()
	for range 1 << 16 {
		m.Put(rng.IntN(1<<20), 0)
	}
	for b.Loop() {
		m.Get(rng.IntN(1 << 20))
	}
}
