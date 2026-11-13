package result_test

import (
	"fmt"
	"strconv"

	"github.com/t1sk3/go-utils/types/result"
)

func Example() {
	double := func(n int) int { return n * 2 }
	for _, s := range []string{"21", "x"} {
		r := result.Of(strconv.Atoi(s)).Map(double)
		fmt.Println(r.IsOk(), r.OrElse(-1))
	}
	// Output:
	// true 42
	// false -1
}
