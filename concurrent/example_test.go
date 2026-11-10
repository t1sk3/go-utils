package concurrent_test

import (
	"fmt"
	"sync"

	"github.com/t1sk3/go-utils/concurrent"
)

func ExampleMap_Merge() {
	var counts concurrent.Map[string, int]
	add := func(a, b int) int { return a + b }

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			counts.Merge("hits", 1, add)
		})
	}
	wg.Wait()

	fmt.Println(counts.Get("hits").MustGet())
	// Output: 100
}

func ExampleMap_ComputeIfAbsent() {
	var cache concurrent.Map[string, int]
	load := func(key string) int {
		fmt.Println("loading", key)
		return len(key)
	}
	fmt.Println(cache.ComputeIfAbsent("gopher", load))
	fmt.Println(cache.ComputeIfAbsent("gopher", load))
	// Output:
	// loading gopher
	// 6
	// 6
}
