# go-utils

Small, dependency-free generic utilities for Go 1.24+.

```bash
go get github.com/t1sk3/go-utils
```

| Package | What it gives you |
|---|---|
| [`stream`](stream) | Lazy, chainable pipelines over slices, iterators and generators |
| [`optional`](optional) | A zero-allocation container for a value that may be absent |

## stream

```go
import "github.com/t1sk3/go-utils/stream"

adults := stream.FromSlice(users).
	Filter(func(u User) bool { return u.Age >= 18 }).
	Sorted(func(a, b User) int { return cmp.Compare(a.Name, b.Name) }).
	Limit(10).
	ToSlice()
```

Streams are **lazy**: intermediate operations only describe the pipeline, and
elements flow through it one at a time when a terminal operation runs. That means
no intermediate slices, and `Limit`, `FindFirst`, `AnyMatch` etc. stop reading the
source as soon as they have their answer, so infinite streams work too:

```go
stream.Iterate(1, func(i int) int { return i * 2 }).Limit(10).ToSlice() // [1 2 4 ... 512]
```

Go methods can't add type parameters, so operations that change the element type
or need `comparable` / `cmp.Ordered` are package functions:

```go
names := stream.Map(stream.FromSlice(users), func(u User) string { return u.Name })
total := stream.Sum(stream.Map(stream.FromSlice(orders), Order.Total))
byTeam := stream.GroupBy(stream.FromSlice(users), func(u User) string { return u.Team })
csv := stream.Join(stream.Distinct(stream.Of("a", "b", "a")), ",") // "a,b"
```

Streams interoperate with `iter.Seq`, so they work with `range` and the standard library:

```go
for k := range stream.Sort(stream.FromSeq(maps.Keys(m))).All() { ... }
```

| | |
|---|---|
| **Sources** | `Of`, `FromSlice`, `FromSeq`, `Range`, `Iterate`, `Generate`, `Concat`, `Empty` |
| **Intermediate methods** | `Filter`, `Map` (same type), `Peek`, `Limit`, `Skip`, `TakeWhile`, `DropWhile`, `Sorted` |
| **Terminal methods** | `ToSlice`, `ForEach`, `Count`, `Reduce`, `Min`, `Max`, `FindFirst`, `AnyMatch`, `AllMatch`, `NoneMatch`, `All` |
| **Generic functions** | `Map`, `FlatMap`, `Distinct`, `DistinctBy`, `Sort`, `Fold`, `Sum`, `Min`, `Max`, `GroupBy`, `ToMap`, `Join` |

## optional

```go
import "github.com/t1sk3/go-utils/optional"

port := optional.OfOk(os.LookupEnv("PORT")).OrElse("8080")

if u, ok := findUser(id).Value(); ok { ... }

name := optional.Map(findUser(id), func(u User) string { return u.Name }).OrElse("anonymous")
```

`Optional[T]` is a plain value type. The zero value is empty, it never allocates,
and it marshals to/from JSON as the value or `null` (use `json:",omitzero"` to drop
empty fields).

## Performance

Apple M5, `go test -bench . -benchmem`, comparing against the previous eager implementation:

| Benchmark | Before | After | Speed-up |
|---|---:|---:|---:|
| Filter → Map → Reduce, 10k ints | 86.4 µs · 9,992 allocs | 32.8 µs · 11 allocs | 2.6× |
| Sorted, 1k ints | 591 µs | 43.9 µs | 13× |
| Distinct, 10k ints | 486 µs · 1.31 MB | 143 µs · 0.65 MB | 3.4× |
| Filter → Limit(10), 1M ints | 2.15 ms · 13.3 MB | 181 ns · 488 B | ~11,900× |
| Filter → FindFirst, 10k ints | 21.5 µs · 128 KB | 40.8 ns · 152 B | ~530× |
| `optional.Of` + `Get` | 9.19 ns · 2 allocs | 1.79 ns · 0 allocs | 5.1× |
