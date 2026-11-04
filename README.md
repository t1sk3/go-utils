# go-utils

[![Go Reference](https://pkg.go.dev/badge/github.com/t1sk3/go-utils.svg)](https://pkg.go.dev/github.com/t1sk3/go-utils)
[![CI](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml/badge.svg)](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml)

Small, dependency-free generic utilities for Go 1.24+.

The repository holds two modules:

| Module | Package | What it gives you |
|---|---|---|
| `github.com/t1sk3/go-utils` | [`stream`](stream) | Lazy, chainable pipelines over slices, iterators and generators |
| | [`future`](future) | `Future[T]`, the typed result of work running on another goroutine, with `All`, `Any` and `Then` |
| | [`concurrent`](concurrent) | `Map[K, V]`, a typed map with atomic `ComputeIfAbsent`, `Compute` and `Merge` |
| `github.com/t1sk3/go-utils/types` | [`optional`](types/optional) | A zero-allocation container for a value that may be absent |
| | [`deque`](types/deque) | `Deque[T]`, a double-ended queue that also works as a FIFO queue or a stack |
| | [`pq`](types/pq) | `Queue[T]`, a priority queue ordered by a comparator |
| | [`pair`](types/pair) | `Pair[A, B]`, two values of possibly different types |
| | [`sorted`](types/sorted) | `Map[K, V]` and `Set[T]`, kept in comparator order with floor, ceiling and range lookups |
| | [`set`](types/set) | `Set[T]`, an unordered collection of distinct values |
| | [`linked`](types/linked) | `Map[K, V]` and `Set[T]` that remember insertion order, which also makes an LRU cache simple |
| | [`constraints`](types/constraints) | `Integer`, `Number`, `Numeric`, `SignedNumber`, `Basic`, ... constraints for generic code |

`types` holds the shared vocabulary and has no dependencies, so your own libraries
can accept or return an `optional.Optional` without pulling in `stream` or anything
else built on top of it:

```bash
go get github.com/t1sk3/go-utils/types   # just the types
go get github.com/t1sk3/go-utils         # the utilities (brings in types)
```

The two modules are versioned separately: tags `vX.Y.Z` release the root module and
`types/vX.Y.Z` release `types`. Both are still at v0, so minor releases may contain
breaking changes until v1.0.0. See [RELEASING.md](RELEASING.md) for how releases are cut.

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

Two-value iterators come in as streams of `pair.Pair` and go back out with `ToSeq2`:

```go
for i, name := range stream.ToSeq2(stream.Enumerate(names)) { ... }
byID := maps.Collect(stream.ToSeq2(stream.Zip(ids, users)))
```

Batching, sliding windows and running totals work on infinite streams too:

```go
for batch := range stream.Chunk(rows, 500).All() { db.InsertMany(batch) }
avg := stream.Map(stream.Window(prices, 7), mean)                // 7-day moving average
balance := stream.Scan(changes, 0, func(t, c int) int { return t + c })
```

Collectors reduce a stream and nest, so a grouping can reduce each group instead of
keeping its elements. They are the counterparts of Java's `Collectors`:

```go
averageAge := stream.Collect(users, stream.GroupingBy(User.Team, stream.Averaging(User.Age)))
activeByTeam := stream.Collect(users, stream.GroupingBy(User.Team,
	stream.Filtering(User.IsActive, stream.Counting[User]())))
```

Common predicates are built in, and `Not`, `And` and `Or` combine them:

```go
stream.Range(0, 10).Filter(stream.IsEven)                    // 0 2 4 6 8
stream.Range(0, 10).Filter(stream.Not(stream.IsEven[int]))   // 1 3 5 7 9
stream.FromSlice(ages).Filter(stream.Between(13, 19))
stream.FromSlice(names).Filter(stream.Not(stream.IsZero[string]))
```

`Parallel` runs the stages after it on several goroutines, until `Sequential` returns to
one-at-a-time processing. Results stay in source order, infinite sources work, and a
panic in a stage comes back on the consuming goroutine. `MapConcurrent` is the shorthand
for a single stage:

```go
pages := stream.MapConcurrent(urls, 8, fetch).ToSlice()
ok := stream.ParallelMap(stream.FromSlice(images).Parallel(runtime.GOMAXPROCS(0)), resize).
	Filter(isValid).
	Sequential().
	Count()
```

Each element costs about half a microsecond of coordination, so this suits I/O and heavy
computation, not cheap functions. See [Performance](#performance) for measurements.

`Comparing`, `Reversed` and `ThenComparing` build comparators for `Sorted`, `Min` and `Max`:

```go
byAge := stream.Comparing(func(u User) int { return u.Age })
byName := stream.Comparing(func(u User) string { return u.Name })
stream.FromSlice(users).Sorted(stream.ThenComparing(stream.Reversed(byAge), byName)) // oldest first, then by name
```

| | |
|---|---|
| **Sources** | `Of`, `FromSlice`, `FromSeq`, `FromSeq2`, `FromMap`, `FromOptional`, `OfNullable`, `Range`, `RangeClosed`, `Iterate`, `IterateWhile`, `Generate`, `Concat`, `Empty` |
| **Intermediate methods** | `Filter`, `Map` (same type), `Peek`, `Limit`, `Skip`, `TakeWhile`, `DropWhile`, `Sorted` |
| **Terminal methods** | `ToSlice`, `ForEach`, `Count`, `Reduce`, `Min`, `Max`, `FindFirst`, `FindLast`, `AnyMatch`, `AllMatch`, `NoneMatch`, `Partition`, `All` |
| **Generic functions** | `Map`, `FlatMap`, `Zip`, `Enumerate`, `Chunk`, `Window`, `Scan`, `Distinct`, `DistinctBy`, `Sort`, `Fold`, `Sum`, `Average`, `Summarize`, `Min`, `Max`, `GroupBy`, `CountBy`, `ToMap`, `ToMapMerge`, `ToSet`, `ToSeq2`, `Join`, `Identity` |
| **Comparators** | `Comparing`, `Reversed`, `ThenComparing`, `NilsFirst`, `NilsLast` |
| **Predicates** | `IsEven`, `IsOdd`, `IsPositive`, `IsNegative`, `IsZero`, `Equal`, `GreaterThan`, `LessThan`, `AtLeast`, `AtMost`, `Between`, `In`, `Not`, `And`, `Or` |
| **Collectors** | `Collect`, `NewCollector`, `IntoSlice`, `IntoSet`, `Joining`, `Counting`, `Summing`, `Averaging`, `MinBy`, `MaxBy`, `Folding`, `GroupingBy`, `PartitioningBy`, `Mapping`, `FlatMapping`, `Filtering`, `CollectingAndThen`, `Teeing` |

## optional

```go
import "github.com/t1sk3/go-utils/types/optional"

port := optional.OfOk(os.LookupEnv("PORT")).OrElse("8080")

if u, ok := findUser(id).Value(); ok { ... }

name := optional.Map(findUser(id), func(u User) string { return u.Name }).OrElse("anonymous")

u, err := findUser(id).OrElseErr(ErrNotFound)
timeout := optional.FromMap(settings, "timeout").OrElse(30)
```

`Optional[T]` is a plain value type. The zero value is empty, it never allocates,
and it marshals to/from JSON as the value or `null` (use `json:",omitzero"` to drop
empty fields).

It also reads nullable database columns: `Optional` implements `sql.Scanner`, so
NULL scans to an empty Optional. Its `Value() (T, bool)` method rules out
`driver.Valuer`, so to write one back use `opt.SQL()`, or declare struct fields as
`optional.Null[T]`, an `Optional` that implements both interfaces:

```go
var email optional.Optional[string]
db.QueryRow("SELECT email FROM users WHERE id = $1", id).Scan(&email)
db.Exec("UPDATE users SET email = $1 WHERE id = $2", email.SQL(), id)
```

## set

```go
import "github.com/t1sk3/go-utils/types/set"

var seen set.Set[string] // the zero value is ready to use
seen.Add("a", "b")

admins := stream.ToSet(stream.Map(stream.FromSlice(users).Filter(User.IsAdmin), User.ID))
missing := required.Difference(granted)
```

`Set` has `Union`, `Intersection`, `Difference`, `SymmetricDifference`,
`IsSubsetOf`, `IsSupersetOf`, `IsDisjoint`, `Equal`, iterates with `All()` and marshals
to/from a JSON array. `Add`, `Remove`, `RemoveIf` and `RetainAll` return how many values
they changed.
Like a map, copies share storage; use `Clone` for an independent copy.

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

`MapConcurrent` against `Map` over 1,000 ints on 10 cores, for functions of different cost:

| Function cost per element | `Map` | `MapConcurrent` | Result |
|---:|---:|---:|---|
| 4 ns | 4.3 µs | 536 µs | 125× slower |
| 0.7 µs | 661 µs | 770 µs | 1.2× slower |
| 2 µs | 2.0 ms | 1.3 ms | 1.6× faster |
| 6.7 µs | 6.7 ms | 2.6 ms | 2.6× faster |
| 67 µs | 67 ms | 14 ms | 4.8× faster |
