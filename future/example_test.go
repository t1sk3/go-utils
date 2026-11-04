package future_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/t1sk3/go-utils/future"
)

func Example() {
	ctx := context.Background()
	name := future.Go(func() (string, error) { return "alice", nil })
	shout := future.Then(name, func(s string) (string, error) { return strings.ToUpper(s), nil })

	got, err := shout.Get(ctx)
	fmt.Println(got, err)
	// Output: ALICE <nil>
}

func ExampleAll() {
	square := func(i int) *future.Future[int] {
		return future.Go(func() (int, error) { return i * i, nil })
	}
	values, err := future.All(context.Background(), square(1), square(2), square(3))
	fmt.Println(values, err)
	// Output: [1 4 9] <nil>
}
