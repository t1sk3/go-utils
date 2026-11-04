package linked_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/linked"
)

func Example() {
	m := linked.NewMap[string, int]()
	m.Put("first", 1)
	m.Put("second", 2)
	m.Put("first", 10) // an existing key keeps its position

	for k, v := range m.All() {
		fmt.Println(k, v)
	}
	// Output:
	// first 10
	// second 2
}

// A Map used as an LRU cache: reads move entries to the back, and the entry at the
// front is the one to evict.
func ExampleMap_MoveToBack() {
	const limit = 2
	cache := linked.NewMap[string, int]()
	get := func(key string) {
		if cache.MoveToBack(key) {
			return
		}
		cache.Put(key, len(key))
		if cache.Len() > limit {
			fmt.Println("evicted", cache.PopFirst().MustGet().First)
		}
	}

	get("a")
	get("b")
	get("a")
	get("c")
	fmt.Println(cache)
	// Output:
	// evicted b
	// Map[a:1 c:1]
}

func ExampleNewSet() {
	fmt.Println(linked.NewSet("b", "a", "b"))
	// Output: Set[b a]
}
