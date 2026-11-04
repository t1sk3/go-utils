package deque_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/deque"
)

func Example() {
	var tasks deque.Deque[string] // the zero value is ready to use
	tasks.PushBack("write", "test")
	tasks.PushFront("plan")

	for !tasks.IsEmpty() {
		fmt.Println(tasks.PopFront().MustGet())
	}
	// Output:
	// plan
	// write
	// test
}

func ExampleDeque_PopBack() {
	stack := deque.Of(1, 2, 3)
	if top, ok := stack.PopBack().Value(); ok {
		fmt.Println(top, stack)
	}
	// Output: 3 Deque[1 2]
}
