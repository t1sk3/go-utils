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

func ExampleZip() {
	names := stream.Of("alice", "bob", "carol")
	ages := stream.Of(31, 17)
	for name, age := range stream.ToSeq2(stream.Zip(names, ages)) {
		fmt.Println(name, age)
	}
	// Output:
	// alice 31
	// bob 17
}

func ExampleChunk() {
	for batch := range stream.Chunk(stream.Range(1, 8), 3).All() {
		fmt.Println(batch)
	}
	// Output:
	// [1 2 3]
	// [4 5 6]
	// [7]
}

func ExampleWindow() {
	prices := stream.Of(10.0, 12.0, 11.0, 15.0)
	movingAvg := stream.Map(stream.Window(prices, 2), func(w []float64) float64 {
		return (w[0] + w[1]) / 2
	})
	fmt.Println(movingAvg.ToSlice())
	// Output: [11 11.5 13]
}

func ExampleScan() {
	balance := stream.Scan(stream.Of(100, -30, 50, -20), 0, func(total, change int) int {
		return total + change
	})
	fmt.Println(balance.ToSlice())
	// Output: [100 70 120 100]
}
