package stream_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/stream"
)

func ExampleGroupingBy() {
	users := stream.Of(user{"alice", 31, true}, user{"bob", 17, true}, user{"carol", 45, false})
	byActive := func(u user) bool { return u.Active }
	age := func(u user) int { return u.Age }

	fmt.Println(stream.Collect(users, stream.GroupingBy(byActive, stream.Summing(age))))
	// Output: map[false:45 true:48]
}

func ExamplePartitioningBy() {
	parts := stream.Collect(stream.Range(0, 5), stream.PartitioningBy(stream.IsEven[int], stream.Counting[int]()))
	fmt.Println(parts[true], parts[false])
	// Output: 3 2
}

func ExampleTeeing() {
	mean := stream.Collect(stream.Of(2, 4, 9), stream.Teeing(
		stream.Summing(func(i int) int { return i }),
		stream.Counting[int](),
		func(sum, count int) int { return sum / count },
	))
	fmt.Println(mean)
	// Output: 5
}
