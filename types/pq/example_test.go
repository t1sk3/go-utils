package pq_test

import (
	"cmp"
	"fmt"

	"github.com/t1sk3/go-utils/types/pq"
)

func Example() {
	type task struct {
		name     string
		priority int
	}
	byPriority := func(a, b task) int { return cmp.Compare(a.priority, b.priority) }

	tasks := pq.New(byPriority, task{"write", 2}, task{"plan", 1})
	tasks.Push(task{"ship", 3})

	for t := range tasks.Drain() {
		fmt.Println(t.name)
	}
	// Output:
	// plan
	// write
	// ship
}

func ExampleQueue_Pop() {
	smallest := pq.New(cmp.Compare[int], 5, 1, 3)
	if v, ok := smallest.Pop().Value(); ok {
		fmt.Println(v)
	}
	// Output: 1
}
