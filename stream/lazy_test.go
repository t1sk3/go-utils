package stream

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestLazy_NothingRunsWithoutTerminal(t *testing.T) {
	calls := 0
	_ = FromSlice([]int{1, 2, 3}).Peek(func(int) { calls++ }).Filter(func(int) bool { calls++; return true })
	if calls != 0 {
		t.Errorf("intermediate operations ran eagerly: %d calls", calls)
	}
}

func TestLazy_LimitStopsPullingFromSource(t *testing.T) {
	pulled := 0
	got := FromSlice([]int{1, 2, 3, 4, 5, 6}).
		Peek(func(int) { pulled++ }).
		Filter(func(i int) bool { return i%2 == 0 }).
		Limit(2).
		ToSlice()
	if !reflect.DeepEqual(got, []int{2, 4}) {
		t.Errorf("got %v, want [2 4]", got)
	}
	if pulled != 4 {
		t.Errorf("pulled %d elements from source, want 4", pulled)
	}
}

func TestLazy_ShortCircuitTerminals(t *testing.T) {
	pulled := 0
	s := FromSlice([]int{1, 2, 3, 4, 5}).Peek(func(int) { pulled++ })
	if !s.AnyMatch(func(i int) bool { return i == 2 }) || pulled != 2 {
		t.Errorf("AnyMatch pulled %d, want 2", pulled)
	}
	pulled = 0
	if s.AllMatch(func(i int) bool { return i < 3 }) || pulled != 3 {
		t.Errorf("AllMatch pulled %d, want 3", pulled)
	}
	pulled = 0
	if v := s.FindFirst().Get(); v != 1 || pulled != 1 {
		t.Errorf("FindFirst = %d after %d pulls, want 1 after 1", v, pulled)
	}
}

func TestInfiniteStreams(t *testing.T) {
	powers := Iterate(1, func(i int) int { return i * 2 }).Limit(5).ToSlice()
	if !reflect.DeepEqual(powers, []int{1, 2, 4, 8, 16}) {
		t.Errorf("Iterate = %v", powers)
	}
	n := 0
	gen := Generate(func() int { n++; return n }).TakeWhile(func(i int) bool { return i <= 3 }).ToSlice()
	if !reflect.DeepEqual(gen, []int{1, 2, 3}) {
		t.Errorf("Generate = %v", gen)
	}
}

func TestReusable(t *testing.T) {
	s := Distinct(Of(3, 1, 3, 2)).Sorted(func(a, b int) int { return a - b })
	first, second := s.ToSlice(), s.ToSlice()
	if !reflect.DeepEqual(first, []int{1, 2, 3}) || !reflect.DeepEqual(first, second) {
		t.Errorf("consuming twice gave %v and %v", first, second)
	}
}

func TestToSlice_DoesNotAliasSource(t *testing.T) {
	src := []int{1, 2, 3}
	out := FromSlice(src).ToSlice()
	out[0] = 99
	if src[0] != 1 {
		t.Errorf("ToSlice result aliases the source slice")
	}
	if got := FromSlice([]int{}).Filter(func(int) bool { return true }).ToSlice(); got == nil {
		t.Errorf("ToSlice returned nil for an empty stream")
	}
}

func TestZeroValueIsEmpty(t *testing.T) {
	var s Stream[int]
	if s.Count() != 0 || s.FindFirst().IsPresent() || len(s.Filter(func(int) bool { return true }).ToSlice()) != 0 {
		t.Errorf("zero Stream is not empty")
	}
}

func TestLimitSkipEdges(t *testing.T) {
	s := Of(1, 2, 3)
	filtered := s.Filter(func(int) bool { return true })
	cases := []struct {
		name string
		got  []int
		want []int
	}{
		{"slice Limit(0)", s.Limit(0).ToSlice(), []int{}},
		{"slice Limit(10)", s.Limit(10).ToSlice(), []int{1, 2, 3}},
		{"slice Skip(10)", s.Skip(10).ToSlice(), []int{}},
		{"seq Limit(0)", filtered.Limit(0).ToSlice(), []int{}},
		{"seq Limit(10)", filtered.Limit(10).ToSlice(), []int{1, 2, 3}},
		{"seq Skip(1)", filtered.Skip(1).ToSlice(), []int{2, 3}},
		{"seq Skip(10)", filtered.Skip(10).ToSlice(), []int{}},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestDropWhile(t *testing.T) {
	got := Of(1, 2, 5, 1, 2).DropWhile(func(i int) bool { return i < 3 }).ToSlice()
	if !reflect.DeepEqual(got, []int{5, 1, 2}) {
		t.Errorf("DropWhile = %v", got)
	}
}

func TestSorted_Stable(t *testing.T) {
	type p struct {
		name string
		age  int
	}
	people := []p{{"a", 30}, {"b", 20}, {"c", 30}, {"d", 20}}
	got := FromSlice(people).Sorted(func(x, y p) int { return x.age - y.age }).ToSlice()
	want := []p{{"b", 20}, {"d", 20}, {"a", 30}, {"c", 30}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sorted = %v, want %v", got, want)
	}
}

func TestSourceConstructors(t *testing.T) {
	if got := Range(2, 6).ToSlice(); !reflect.DeepEqual(got, []int{2, 3, 4, 5}) {
		t.Errorf("Range = %v", got)
	}
	if got := Range(5, 2).Count(); got != 0 {
		t.Errorf("Range with end < start has %d elements", got)
	}
	if got := Range[int8](-128, 127).Count(); got != 255 {
		t.Errorf("Range over full int8 has %d elements, want 255", got)
	}
	keys := Sort(FromSeq(maps.Keys(map[string]int{"b": 1, "a": 2}))).ToSlice()
	if !reflect.DeepEqual(keys, []string{"a", "b"}) {
		t.Errorf("FromSeq(maps.Keys) = %v", keys)
	}
	if got := Concat(Of(1), Empty[int](), Range(2, 4)).ToSlice(); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("Concat = %v", got)
	}
	if got := slices.Collect(Of(1, 2).All()); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Errorf("All = %v", got)
	}
}

func TestGenericFunctions(t *testing.T) {
	words := Of("go", "is", "fun", "go")

	if got := FlatMap(Of(1, 2), func(i int) Stream[int] { return Of(i, i*10) }).ToSlice(); !reflect.DeepEqual(got, []int{1, 10, 2, 20}) {
		t.Errorf("FlatMap = %v", got)
	}
	if got := Fold(words, 0, func(n int, w string) int { return n + len(w) }); got != 9 {
		t.Errorf("Fold = %d, want 9", got)
	}
	if got := Sum(Of(1.5, 2.5)); got != 4 {
		t.Errorf("Sum = %v, want 4", got)
	}
	if got := Min(words).Get(); got != "fun" {
		t.Errorf("Min = %q", got)
	}
	if got := Max(Of(3, 9, 2)).Get(); got != 9 {
		t.Errorf("Max = %d", got)
	}
	if Min(Empty[int]()).IsPresent() {
		t.Errorf("Min of empty stream is present")
	}
	if got := Join(Distinct(words), ","); got != "go,is,fun" {
		t.Errorf("Join(Distinct) = %q", got)
	}
	if got := DistinctBy(words, func(w string) int { return len(w) }).ToSlice(); !reflect.DeepEqual(got, []string{"go", "fun"}) {
		t.Errorf("DistinctBy = %v", got)
	}
	groups := GroupBy(words, func(w string) int { return len(w) })
	if !reflect.DeepEqual(groups, map[int][]string{2: {"go", "is", "go"}, 3: {"fun"}}) {
		t.Errorf("GroupBy = %v", groups)
	}
	m := ToMap(words, strings.ToUpper, func(w string) int { return len(w) })
	if !reflect.DeepEqual(m, map[string]int{"GO": 2, "IS": 2, "FUN": 3}) {
		t.Errorf("ToMap = %v", m)
	}
	if got := words.Map(strings.ToUpper).Limit(2).ToSlice(); !reflect.DeepEqual(got, []string{"GO", "IS"}) {
		t.Errorf("Stream.Map = %v", got)
	}
}

func TestNilFunctionPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != "stream: Filter requires a non-nil function" {
			t.Errorf("unexpected panic value %v", r)
		}
	}()
	Of(1).Filter(nil)
}
