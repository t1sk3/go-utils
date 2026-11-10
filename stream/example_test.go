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

	names := stream.FromSlice(users).
		Filter(func(u user) bool { return u.Active && u.Age >= 18 }).
		Sorted(func(a, b user) int { return a.Age - b.Age }).
		Map(func(u user) string { return u.Name }).
		ToSlice()

	fmt.Println(names)
	// Output: [dave alice]
}

func ExampleComparing() {
	users := []user{{Name: "alice", Age: 31}, {Name: "bob", Age: 17}, {Name: "carol", Age: 31}}
	byAge := stream.Comparing(func(u user) int { return u.Age })
	byName := stream.Comparing(func(u user) string { return u.Name })

	// Oldest first, with ties in reverse alphabetical order.
	sorted := stream.FromSlice(users).Sorted(stream.Reversed(stream.ThenComparing(byAge, byName)))
	fmt.Println(sorted.Map(func(u user) string { return u.Name }).ToSlice())
	// Output: [carol alice bob]
}

func ExampleStream_CountBy() {
	words := stream.Of("apple", "avocado", "banana")
	counts := words.CountBy(func(w string) string { return w[:1] })
	fmt.Println(counts)
	// Output: map[a:2 b:1]
}

func ExampleStream_ToMapMerge() {
	type order struct {
		Customer string
		Amount   int
	}
	orders := stream.Of(order{"ann", 5}, order{"ben", 3}, order{"ann", 2})
	totals := orders.ToMapMerge(
		func(o order) string { return o.Customer },
		func(o order) int { return o.Amount },
		func(a, b int) int { return a + b })
	fmt.Println(totals)
	// Output: map[ann:7 ben:3]
}

func ExampleRangeClosed() {
	fmt.Println(stream.RangeClosed(1, 5).ToSlice())
	// Output: [1 2 3 4 5]
}

func ExampleAverage() {
	fmt.Println(stream.Average(stream.Of(1, 2, 6)).Get())
	// Output: 3
}

func ExampleIterate() {
	// Infinite streams are fine as long as something bounds them.
	fmt.Println(stream.Iterate(1, func(i int) int { return i * 3 }).Limit(5).ToSlice())
	// Output: [1 3 9 27 81]
}

func ExampleStream_All() {
	for w := range stream.Of("a", "b", "c").Map(strings.ToUpper).All() {
		fmt.Print(w)
	}
	// Output: ABC
}

func ExampleStream_Zip() {
	names := stream.Of("alice", "bob", "carol")
	ages := stream.Of(31, 17)
	for name, age := range stream.ToSeq2(names.Zip(ages)) {
		fmt.Println(name, age)
	}
	// Output:
	// alice 31
	// bob 17
}

func ExampleStream_Chunk() {
	for batch := range stream.Range(1, 8).Chunk(3).All() {
		fmt.Println(batch)
	}
	// Output:
	// [1 2 3]
	// [4 5 6]
	// [7]
}

func ExampleStream_Window() {
	prices := stream.Of(10.0, 12.0, 11.0, 15.0)
	movingAvg := prices.Window(2).Map(func(w []float64) float64 {
		return (w[0] + w[1]) / 2
	})
	fmt.Println(movingAvg.ToSlice())
	// Output: [11 11.5 13]
}

func ExampleStream_Scan() {
	balance := stream.Of(100, -30, 50, -20).Scan(0, func(total, change int) int {
		return total + change
	})
	fmt.Println(balance.ToSlice())
	// Output: [100 70 120 100]
}

func ExampleStream_FindLast() {
	lastEven := stream.Range(0, 10).Filter(stream.IsEven).FindLast()
	fmt.Println(lastEven.Get())
	// Output: 8
}

func ExampleNot() {
	evens := stream.Range(0, 10).Filter(stream.IsEven)
	odds := stream.Range(0, 10).Filter(stream.Not(stream.IsEven[int]))
	fmt.Println(evens.ToSlice(), odds.ToSlice())
	// Output: [0 2 4 6 8] [1 3 5 7 9]
}

func ExampleAnd() {
	teens := stream.And(stream.AtLeast(13), stream.AtMost(19))
	fmt.Println(stream.Of(9, 13, 17, 21).Filter(teens).ToSlice())
	// Output: [13 17]
}

func ExampleIterateWhile() {
	powers := stream.IterateWhile(1, func(i int) bool { return i < 100 }, func(i int) int { return i * 3 })
	fmt.Println(powers.ToSlice())
	// Output: [1 3 9 27 81]
}

func ExampleSummarize() {
	s := stream.Summarize(stream.Of(3, 1, 2))
	fmt.Println(s.Count, s.Sum, s.Min, s.Max, s.Average)
	// Output: 3 6 1 3 2
}

func ExampleIdentity() {
	lengths := stream.Of("go", "java").ToMap(stream.Identity, func(w string) int { return len(w) })
	fmt.Println(lengths)
	// Output: map[go:2 java:4]
}

func ExampleStream_Map() {
	// Map can change the element type, so the pipeline keeps chaining.
	labels := stream.Range(1, 4).
		Map(func(i int) string { return strings.Repeat("*", i) }).
		Map(func(s string) string { return "[" + s + "]" }).
		ToSlice()
	fmt.Println(labels)
	// Output: [[*] [**] [***]]
}

func ExampleStream_Fold() {
	words := stream.Of("go", "rust", "zig")
	fmt.Println(words.Fold(0, func(n int, w string) int { return n + len(w) }))
	// Output: 9
}

func ExampleStream_GroupBy() {
	byLength := stream.Of("go", "rust", "zig", "c", "java").GroupBy(func(w string) int { return len(w) })
	fmt.Println(byLength[1], byLength[2], byLength[3], byLength[4])
	// Output: [c] [go] [zig] [rust java]
}
