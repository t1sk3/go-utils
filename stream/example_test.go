package stream_test

import (
	"fmt"
	"strings"

	"github.com/t1sk3/go-utils/stream"
)

type user struct {
	Name   string
	Age    int
	Active bool
}

func Example() {
	users := []user{
		{"alice", 31, true},
		{"bob", 17, true},
		{"carol", 45, false},
		{"dave", 25, true},
	}

	names := stream.Map(
		stream.FromSlice(users).
			Filter(func(u user) bool { return u.Active && u.Age >= 18 }).
			Sorted(func(a, b user) int { return a.Age - b.Age }),
		func(u user) string { return u.Name },
	).ToSlice()

	fmt.Println(names)
	// Output: [dave alice]
}

func ExampleIterate() {
	// Infinite streams are fine as long as something bounds them.
	fmt.Println(stream.Iterate(1, func(i int) int { return i * 3 }).Limit(5).ToSlice())
	// Output: [1 3 9 27 81]
}

func ExampleGroupBy() {
	byLen := stream.GroupBy(stream.Of("go", "rust", "zig", "java"), func(s string) int { return len(s) })
	fmt.Println(byLen[3], byLen[4])
	// Output: [zig] [rust java]
}

func ExampleStream_All() {
	for w := range stream.Of("a", "b", "c").Map(strings.ToUpper).All() {
		fmt.Print(w)
	}
	// Output: ABC
}
