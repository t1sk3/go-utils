package stream

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

func TestSimpleCollectors(t *testing.T) {
	nums := Of(3, 1, 2)
	double := func(i int) int { return i * 2 }

	if got := Collect(nums, IntoSlice[int]()); !slices.Equal(got, []int{3, 1, 2}) {
		t.Errorf("IntoSlice = %v", got)
	}
	if got := Collect(Empty[int](), IntoSlice[int]()); got == nil || len(got) != 0 {
		t.Errorf("IntoSlice of nothing = %#v, want an empty non-nil slice", got)
	}
	if got := Collect(Of(1, 2, 1), IntoSet[int]()); got.Len() != 2 || !got.Has(1) || !got.Has(2) {
		t.Errorf("IntoSet = %v", got)
	}
	if got := Collect(Of("a", "b", "c"), Joining[string](",")); got != "a,b,c" {
		t.Errorf("Joining = %q", got)
	}
	if got := Collect(Empty[string](), Joining[string](",")); got != "" {
		t.Errorf("Joining nothing = %q", got)
	}
	if got := Collect(nums, Counting[int]()); got != 3 {
		t.Errorf("Counting = %d", got)
	}
	if got := Collect(nums, Summing(double)); got != 12 {
		t.Errorf("Summing = %d", got)
	}
	if got := Collect(nums, Averaging(double)); got != optional.Of(4.0) {
		t.Errorf("Averaging = %v", got)
	}
	if got := Collect(Empty[int](), Averaging(double)); got.IsPresent() {
		t.Errorf("Averaging nothing = %v", got)
	}
	if got := Collect(nums, Folding("", func(acc string, i int) string { return acc + strings.Repeat("x", i) })); got != "xxxxxx" {
		t.Errorf("Folding = %q", got)
	}
}

func TestMinByAndMaxByKeepFirstOfTies(t *testing.T) {
	alice, bob, carol := person{"alice", 30}, person{"bob", 20}, person{"carol", 30}
	people := Of(alice, bob, carol, person{"dave", 20})
	byAge := Comparing(personAge)
	if got := Collect(people, MinBy(byAge)); got != optional.Of(bob) {
		t.Errorf("MinBy = %v", got)
	}
	if got := Collect(people, MaxBy(byAge)); got != optional.Of(alice) {
		t.Errorf("MaxBy = %v", got)
	}
	if got := Collect(Empty[person](), MinBy(byAge)); got.IsPresent() {
		t.Errorf("MinBy of nothing = %v", got)
	}
	// They must agree with the Stream methods of the same purpose.
	if got, want := Collect(people, MinBy(byAge)), people.Min(byAge); got != want {
		t.Errorf("MinBy = %v, Stream.Min = %v", got, want)
	}
	if got, want := Collect(people, MaxBy(byAge)), people.Max(byAge); got != want {
		t.Errorf("MaxBy = %v, Stream.Max = %v", got, want)
	}
}

func TestGroupingBy(t *testing.T) {
	alice, bob, carol := person{"alice", 30}, person{"bob", 20}, person{"carol", 30}
	people := Of(alice, bob, carol)

	if got, want := Collect(people, GroupingBy(personAge, IntoSlice[person]())), GroupBy(people, personAge); !reflect.DeepEqual(got, want) {
		t.Errorf("IntoSlice: got %v, want the same as GroupBy: %v", got, want)
	}
	if got, want := Collect(people, GroupingBy(personAge, Counting[person]())), CountBy(people, personAge); !reflect.DeepEqual(got, want) {
		t.Errorf("Counting: got %v, want the same as CountBy: %v", got, want)
	}
	got := Collect(people, GroupingBy(personAge, Mapping(personName, Joining[string]("+"))))
	if want := map[int]string{30: "alice+carol", 20: "bob"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Mapping: got %v, want %v", got, want)
	}
	if got := Collect(Empty[person](), GroupingBy(personAge, Counting[person]())); got == nil || len(got) != 0 {
		t.Errorf("empty = %#v, want an empty non-nil map", got)
	}
}

func TestNestedGroupingBy(t *testing.T) {
	people := Of(person{"alice", 30}, person{"anna", 30}, person{"bob", 30}, person{"bert", 20})
	firstLetter := func(p person) string { return p.name[:1] }
	got := Collect(people, GroupingBy(personAge, GroupingBy(firstLetter, Counting[person]())))
	want := map[int]map[string]int{30: {"a": 2, "b": 1}, 20: {"b": 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPartitioningBy(t *testing.T) {
	got := Collect(Of(1, 2, 3), PartitioningBy(IsEven[int], IntoSlice[int]()))
	want := map[bool][]int{true: {2}, false: {1, 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	counts := Collect(Of(1, 3), PartitioningBy(IsEven[int], Counting[int]()))
	if n, ok := counts[true]; !ok || n != 0 || counts[false] != 2 {
		t.Errorf("a side with no elements is missing: %v", counts)
	}
}

func TestFilteringKeepsEmptyGroups(t *testing.T) {
	adults := func(p person) bool { return p.age >= 18 }
	people := Of(person{"alice", 30}, person{"tom", 10})
	got := Collect(people, GroupingBy(personName, Filtering(adults, Counting[person]())))
	if want := map[string]int{"alice": 1, "tom": 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFlatMapping(t *testing.T) {
	words := func(s string) Stream[string] { return FromSlice(strings.Fields(s)) }
	got := Collect(Of("a b", "", "c"), FlatMapping(words, IntoSlice[string]()))
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
}

func TestCollectingAndThen(t *testing.T) {
	got := Collect(Of(3, 1, 2), CollectingAndThen(IntoSlice[int](), func(s []int) []int {
		slices.Sort(s)
		return s
	}))
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("got %v", got)
	}
}

func TestTeeing(t *testing.T) {
	got := Collect(Of(1, 2, 3), Teeing(Counting[int](), Summing(identityInt), pair.Of[int, int]))
	if want := pair.Of(3, 6); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := Collect(Of(3, 1), Teeing(MinBy(cmp.Compare[int]), MaxBy(cmp.Compare[int]), func(lo, hi optional.Optional[int]) int {
		return hi.MustGet() - lo.MustGet()
	})); got != 2 {
		t.Errorf("spread = %d", got)
	}
}

func identityInt(i int) int { return i }

func TestNewCollector(t *testing.T) {
	longest := NewCollector(
		func() *strings.Builder { return &strings.Builder{} },
		func(b *strings.Builder, s string) *strings.Builder {
			if len(s) > b.Len() {
				b.Reset()
				b.WriteString(s)
			}
			return b
		},
		(*strings.Builder).String,
	)
	if got := Collect(Of("a", "ccc", "bb"), longest); got != "ccc" {
		t.Errorf("Collect = %q", got)
	}
	// Each run needs its own accumulation, including nested runs for each group.
	byLen := GroupingBy(func(s string) int { return len(s) % 2 }, longest)
	want := map[int]string{1: "ccc", 0: "bb"}
	if got := Collect(Of("a", "ccc", "bb"), byLen); !reflect.DeepEqual(got, want) {
		t.Errorf("grouped = %v, want %v", got, want)
	}
}

func TestCollectorsAreReusable(t *testing.T) {
	collectors := map[string]Collector[int, int]{
		"Counting":  Counting[int](),
		"Summing":   Summing(identityInt),
		"Folding":   Folding(0, func(a, v int) int { return a + v }),
		"Mapping":   Mapping(identityInt, Summing(identityInt)),
		"Filtering": Filtering(func(int) bool { return true }, Summing(identityInt)),
		"Teeing":    Teeing(Summing(identityInt), Counting[int](), func(a, _ int) int { return a }),
	}
	for name, c := range collectors {
		a, b := Collect(Of(1, 1), c), Collect(Of(1, 1), c)
		if a != b {
			t.Errorf("%s: first run %d, second run %d", name, a, b)
		}
	}
}

func TestCollectOnLimitedInfiniteStream(t *testing.T) {
	got := Collect(Iterate(1, func(i int) int { return i + 1 }).Limit(3), Summing(identityInt))
	if got != 6 {
		t.Errorf("got %d, want 6", got)
	}
}

func TestCollectorsRejectNil(t *testing.T) {
	var zero Collector[int, int]
	tests := map[string]func(){
		"Collect zero value":   func() { Collect(Of(1), zero) },
		"NewCollector":         func() { NewCollector[int, int, int](nil, nil, nil) },
		"Summing":              func() { Summing[int, int](nil) },
		"Averaging":            func() { Averaging[int, int](nil) },
		"MinBy":                func() { MinBy[int](nil) },
		"MaxBy":                func() { MaxBy[int](nil) },
		"Folding":              func() { Folding[int, int](0, nil) },
		"GroupingBy key":       func() { GroupingBy[int, int](nil, Counting[int]()) },
		"GroupingBy down":      func() { GroupingBy(identityInt, zero) },
		"PartitioningBy pred":  func() { PartitioningBy(nil, Counting[int]()) },
		"PartitioningBy down":  func() { PartitioningBy(IsEven[int], zero) },
		"Mapping mapper":       func() { Mapping[int, int](nil, Counting[int]()) },
		"Mapping down":         func() { Mapping(identityInt, zero) },
		"FlatMapping mapper":   func() { FlatMapping[int, int](nil, Counting[int]()) },
		"FlatMapping down":     func() { FlatMapping(func(i int) Stream[int] { return Of(i) }, zero) },
		"Filtering pred":       func() { Filtering(nil, Counting[int]()) },
		"Filtering down":       func() { Filtering(IsEven[int], zero) },
		"CollectingAndThen f":  func() { CollectingAndThen[int, int, int](Counting[int](), nil) },
		"CollectingAndThen dn": func() { CollectingAndThen(zero, identityInt) },
		"Teeing merge":         func() { Teeing[int, int, int, int](Counting[int](), Counting[int](), nil) },
		"Teeing first":         func() { Teeing(zero, Counting[int](), func(a, b int) int { return a }) },
		"Teeing second":        func() { Teeing(Counting[int](), zero, func(a, b int) int { return a }) },
	}
	for name, f := range tests {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			f()
		}()
	}
}

func BenchmarkGroupingByCounting(b *testing.B) {
	values := Range(0, 10_000)
	key := func(i int) int { return i % 100 }
	collector := GroupingBy(key, Counting[int]())
	for b.Loop() {
		Collect(values, collector)
	}
}

func BenchmarkCountBy(b *testing.B) {
	values := Range(0, 10_000)
	key := func(i int) int { return i % 100 }
	for b.Loop() {
		CountBy(values, key)
	}
}
