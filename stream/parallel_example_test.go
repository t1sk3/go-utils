package stream_test

import (
	"fmt"
	"strings"

	"github.com/t1sk3/go-utils/stream"
)

func ExampleMapConcurrent() {
	urls := stream.Of("a.example", "b.example", "c.example")
	fetch := func(url string) string { return strings.ToUpper(url) } // Stands in for a slow call.

	// Results come back in source order, however long each call takes.
	fmt.Println(stream.MapConcurrent(urls, 2, fetch).ToSlice())
	// Output: [A.EXAMPLE B.EXAMPLE C.EXAMPLE]
}

func ExampleStream_Parallel() {
	squares := stream.Range(0, 6).
		Parallel(3).
		Filter(stream.IsEven[int]).
		Map(func(i int) int { return i * i }).
		Sequential()
	fmt.Println(stream.Sum(squares))
	// Output: 20
}

func ExampleParallelMap() {
	labels := stream.ParallelMap(stream.Of(1, 2).Parallel(2), func(i int) string {
		return fmt.Sprintf("#%d", i)
	})
	fmt.Println(labels.ToSlice())
	// Output: [#1 #2]
}

func ExampleParallelStream_Map() {
	labels := stream.Of(1, 2, 3).
		Parallel(2).
		Map(func(i int) string { return fmt.Sprintf("#%d", i) }).
		ToSlice()
	fmt.Println(labels)
	// Output: [#1 #2 #3]
}
