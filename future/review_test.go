package future

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNilFuturePanicsOnCaller(t *testing.T) {
	for name, call := range map[string]func(){
		"All": func() { All(context.Background(), Completed(1), nil) },
		"Any": func() { Any(context.Background(), Completed(1), nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				msg, _ := recover().(string)
				if !strings.Contains(msg, name) || !strings.Contains(msg, "index 1") {
					t.Errorf("panic = %q, want one naming %s and index 1", msg, name)
				}
			}()
			call()
		})
	}
}

// As with Get, an outcome that is already there is returned even if ctx has ended.
func TestReadyOutcomeBeatsEndedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	never := Go(func() (int, error) { select {} })

	for range 100 {
		values, err := All(ctx, Completed(1), Completed(2))
		if err != nil || len(values) != 2 || values[0] != 1 || values[1] != 2 {
			t.Fatalf("All of completed futures = %v, %v", values, err)
		}
		if _, err := All(ctx, never, Failed[int](errBoom)); !errors.Is(err, errBoom) {
			t.Fatalf("All with a failed future = %v, want errBoom", err)
		}
		if v, err := Any(ctx, never, Failed[int](errBoom), Completed(7)); err != nil || v != 7 {
			t.Fatalf("Any with a completed future = %v, %v", v, err)
		}
	}
	if _, err := All(ctx, never, Completed(1)); !errors.Is(err, context.Canceled) {
		t.Errorf("All with a pending future = %v, want context.Canceled", err)
	}
	if _, err := Any(ctx, never, Failed[int](errBoom)); !errors.Is(err, context.Canceled) {
		t.Errorf("Any with a pending future = %v, want context.Canceled", err)
	}
}

func TestAllReportsFirstFailureInOrder(t *testing.T) {
	other := errors.New("other")
	if _, err := All(context.Background(), Completed(1), Failed[int](other), Failed[int](errBoom)); err != other {
		t.Errorf("All = %v, want the first failed future's error", err)
	}
}

func TestPanicErrorMessageHasNoStack(t *testing.T) {
	_, err := Go(func() (int, error) { panic("boom") }).Get(context.Background())
	pe, ok := errors.AsType[*PanicError](err)
	if !ok {
		t.Fatalf("error = %v", err)
	}
	if got, want := err.Error(), "future: function panicked: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !strings.Contains(string(pe.Stack), "review_test.go") {
		t.Errorf("Stack does not show where the panic came from:\n%s", pe.Stack)
	}
}
