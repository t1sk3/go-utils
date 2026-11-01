package stream

import (
	"reflect"
	"testing"
)

func TestPredicates(t *testing.T) {
	nums := Range(-3, 4) // -3 .. 3
	tests := []struct {
		name string
		got  []int
		want []int
	}{
		{"IsEven", nums.Filter(IsEven).ToSlice(), []int{-2, 0, 2}},
		{"IsOdd", nums.Filter(IsOdd).ToSlice(), []int{-3, -1, 1, 3}},
		{"Not(IsEven)", nums.Filter(Not(IsEven[int])).ToSlice(), []int{-3, -1, 1, 3}},
		{"IsPositive", nums.Filter(IsPositive).ToSlice(), []int{1, 2, 3}},
		{"IsNegative", nums.Filter(IsNegative).ToSlice(), []int{-3, -2, -1}},
		{"IsZero", nums.Filter(IsZero).ToSlice(), []int{0}},
		{"Equal", nums.Filter(Equal(2)).ToSlice(), []int{2}},
		{"GreaterThan", nums.Filter(GreaterThan(1)).ToSlice(), []int{2, 3}},
		{"LessThan", nums.Filter(LessThan(-1)).ToSlice(), []int{-3, -2}},
		{"AtLeast", nums.Filter(AtLeast(2)).ToSlice(), []int{2, 3}},
		{"AtMost", nums.Filter(AtMost(-2)).ToSlice(), []int{-3, -2}},
		{"Between", nums.Filter(Between(-1, 1)).ToSlice(), []int{-1, 0, 1}},
		{"In", nums.Filter(In(3, -3, 7)).ToSlice(), []int{-3, 3}},
		{"And", nums.Filter(And(IsEven[int], IsPositive[int])).ToSlice(), []int{2}},
		{"Or", nums.Filter(Or(IsZero[int], GreaterThan(2))).ToSlice(), []int{0, 3}},
		{"And()", nums.Filter(And[int]()).ToSlice(), []int{-3, -2, -1, 0, 1, 2, 3}},
		{"Or()", nums.Filter(Or[int]()).ToSlice(), []int{}},
	}
	for _, tt := range tests {
		if len(tt.got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(tt.got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestPredicatesNamedTypes(t *testing.T) {
	type celsius float64
	temps := Of[celsius](-5, 0, 21.5, 30)
	if got := temps.Filter(IsNegative).ToSlice(); !reflect.DeepEqual(got, []celsius{-5}) {
		t.Errorf("IsNegative on named float: got %v", got)
	}
	if got := Of("", "a", "").Filter(Not(IsZero[string])).ToSlice(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("Not(IsZero) on strings: got %v", got)
	}
}

func TestAndShortCircuits(t *testing.T) {
	called := false
	never := func(int) bool { called = true; return true }
	if And(IsEven[int], never)(3) {
		t.Error("And(IsEven, ...)(3) = true, want false")
	}
	if Or(IsOdd[int], never)(3) != true {
		t.Error("Or(IsOdd, ...)(3) = false, want true")
	}
	if called {
		t.Error("And/Or evaluated a predicate after the result was decided")
	}
}

func TestCombinatorsPanicOnNil(t *testing.T) {
	for name, f := range map[string]func(){
		"Not": func() { Not[int](nil) },
		"And": func() { And(IsEven[int], nil) },
		"Or":  func() { Or[int](nil) },
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
