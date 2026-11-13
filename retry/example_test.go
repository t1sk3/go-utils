package retry_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/t1sk3/go-utils/retry"
)

func ExampleDo() {
	calls := 0
	fetch := func(context.Context) (string, error) {
		calls++
		if calls < 3 {
			return "", errors.New("server busy")
		}
		return "page", nil
	}
	page, err := retry.Do(context.Background(), retry.Policy{Attempts: 5}, fetch)
	fmt.Println(page, err, calls)
	// Output: page <nil> 3
}
