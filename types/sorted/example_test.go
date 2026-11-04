package sorted_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/sorted"
)

func Example() {
	versions := sorted.NewOrderedMap[int, string]()
	versions.Put(10, "first release")
	versions.Put(30, "rewrite")
	versions.Put(20, "bug fixes")

	// Find the release in effect at build 25.
	if e, ok := versions.Floor(25).Value(); ok {
		fmt.Println(e.First, e.Second)
	}

	for build, name := range versions.Range(10, 30) {
		fmt.Println(build, name)
	}
	// Output:
	// 20 bug fixes
	// 10 first release
	// 20 bug fixes
}

func ExampleNewOrderedSet() {
	s := sorted.NewOrderedSet(3, 1, 2, 3)
	fmt.Println(s, s.First().MustGet(), s.Higher(2).MustGet())
	// Output: Set[1 2 3] 1 3
}
