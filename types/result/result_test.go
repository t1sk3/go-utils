package result

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

var errBoom = errors.New("boom")

func TestConstructors(t *testing.T) {
	var zero Result[int]
	if !zero.IsOk() || zero.IsErr() || zero.OrElse(9) != 0 {
		t.Errorf("the zero value is not a success holding zero: %v", zero)
	}
	if v, err := Ok(5).Get(); v != 5 || err != nil {
		t.Errorf("Ok(5).Get() = %v, %v", v, err)
	}
	if r := Err[int](errBoom); !r.IsErr() || r.IsOk() || r.Err() != errBoom {
		t.Errorf("Err(errBoom) = %v", r)
	}
	if v, err := Of(5, errBoom).Get(); v != 0 || err != errBoom {
		t.Errorf("Of(5, errBoom).Get() = %v, %v, want the value dropped", v, err)
	}
	if r := Of(strconv.Atoi("12")); r.MustGet() != 12 || r.Err() != nil {
		t.Errorf("Of(Atoi) = %v", r)
	}
	defer func() {
		if recover() == nil {
			t.Errorf("Err(nil) did not panic")
		}
	}()
	Err[int](nil)
}

func TestAccessors(t *testing.T) {
	ok, bad := Ok(5), Err[int](errBoom)
	if ok.OrElse(1) != 5 || bad.OrElse(1) != 1 {
		t.Errorf("OrElse is wrong")
	}
	fallback := func(err error) int { return len(err.Error()) }
	if ok.OrElseGet(fallback) != 5 || bad.OrElseGet(fallback) != 4 {
		t.Errorf("OrElseGet is wrong")
	}
	if v, present := ok.Optional().Value(); !present || v != 5 {
		t.Errorf("Optional of a success = %v, %v", v, present)
	}
	if bad.Optional().IsPresent() {
		t.Errorf("Optional of a failure is present")
	}
	if ok.String() != "Ok(5)" || bad.String() != "Err(boom)" {
		t.Errorf("String() = %q, %q", ok, bad)
	}
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, errBoom) {
			t.Errorf("MustGet panicked with %v, want an error wrapping errBoom", err)
		}
	}()
	bad.MustGet()
}

func TestTransformations(t *testing.T) {
	ok, bad := Ok(5), Err[int](errBoom)
	if got := ok.Map(strconv.Itoa); got.MustGet() != "5" {
		t.Errorf("Map of a success = %v", got)
	}
	if got := bad.Map(strconv.Itoa); got.Err() != errBoom {
		t.Errorf("Map of a failure = %v", got)
	}
	if got := Ok("7").FlatMap(strconv.Atoi); got.MustGet() != 7 {
		t.Errorf("FlatMap = %v", got)
	}
	if got := Ok("x").FlatMap(strconv.Atoi); got.IsOk() {
		t.Errorf("FlatMap with a failing function = %v", got)
	}
	called := false
	if got := Err[string](errBoom).FlatMap(func(s string) (int, error) { called = true; return 0, nil }); got.Err() != errBoom || called {
		t.Errorf("FlatMap of a failure = %v, called = %v", got, called)
	}
	wrap := func(err error) error { return fmt.Errorf("loading: %w", err) }
	if got := bad.MapErr(wrap); got.Err().Error() != "loading: boom" || !errors.Is(got.Err(), errBoom) {
		t.Errorf("MapErr = %v", got)
	}
	if got := ok.MapErr(wrap); got != ok {
		t.Errorf("MapErr of a success = %v", got)
	}
	if got := bad.MapErr(func(error) error { return nil }); !got.IsOk() {
		t.Errorf("MapErr to nil = %v", got)
	}
}
