# go-utils

[![Go Reference](https://pkg.go.dev/badge/github.com/t1sk3/go-utils.svg)](https://pkg.go.dev/github.com/t1sk3/go-utils)
[![CI](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml/badge.svg)](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml)

Small, dependency-free generic utilities. Requires Go 1.27 or later.

| Package | What it gives you |
|---|---|
| [`stream`](stream) | Lazy, chainable pipelines over slices, iterators and generators |
| [`future`](future) | `Future[T]`, the typed result of work running on another goroutine, with `All`, `Any` and `Then` |
| [`concurrent`](concurrent) | `Map[K, V]`, a typed map with atomic `ComputeIfAbsent`, `Compute` and `Merge` |
| [`optional`](types/optional) | A zero-allocation container for a value that may be absent |
| [`deque`](types/deque) | `Deque[T]`, a double-ended queue that also works as a FIFO queue or a stack |
| [`pq`](types/pq) | `Queue[T]`, a priority queue ordered by a comparator |
| [`pair`](types/pair) | `Pair[A, B]`, two values of possibly different types |
| [`sorted`](types/sorted) | `Map[K, V]` and `Set[T]`, kept in comparator order with floor, ceiling and range lookups |
| [`set`](types/set) | `Set[T]`, an unordered collection of distinct values |
| [`linked`](types/linked) | `Map[K, V]` and `Set[T]` that remember insertion order, which also makes an LRU cache simple |
| [`vector`](types/vector) | `VecN[T]` of any dimension, plus `Vec2[T]` and `Vec3[T]` with `X`, `Y`, `Z` fields |
| [`constraints`](types/constraints) | `Integer`, `Number`, `Numeric`, `SignedNumber`, `Basic`, ... constraints for generic code |

Everything is one module, so a single `go get` gives you all packages:

```bash
go get github.com/t1sk3/go-utils
```

Only the packages you import are compiled into your program. `types` and its
subpackages have no dependencies on the rest of the repo, so your own libraries can
accept or return an `optional.Optional` without depending on `stream`.

Releases are tagged `vX.Y.Z`. The module is still at v0, so minor releases may contain
breaking changes until v1.0.0. See [RELEASING.md](RELEASING.md) for how releases are cut.

### Upgrading to v0.6.1

v0.6.1 removes the package functions that v0.6.0 kept next to the new generic methods.
Rewrite each call with its first argument as the receiver:

| Removed | Use instead |
|---|---|
| `stream.Map(s, f)`, `stream.FlatMap`, `stream.DistinctBy`, `stream.Fold`, `stream.Scan`, `stream.GroupBy`, `stream.CountBy`, `stream.ToMap`, `stream.ToMapMerge`, `stream.Collect`, `stream.MapConcurrent` | `s.Map(f)`, `s.FlatMap(…)`, … |
| `stream.ParallelMap(p, f)` | `p.Map(f)` |
| `optional.Map(o, f)`, `optional.FlatMap(o, f)` | `o.Map(f)`, `o.FlatMap(f)` |
| `future.Then(fut, f)` | `fut.Then(f)` |
| `stream.FromArray(s)` (already deprecated) | `stream.FromSlice(s)` |

With explicit type arguments, drop the first one, which the receiver now supplies:
`stream.Map[int, string](s, nil)` becomes `s.Map[string](nil)`.

### Older versions

Versions up to v0.5.0 shipped `types` as a separate module
(`github.com/t1sk3/go-utils/types`). If your `go.mod` still requires it, remove it with
`go get github.com/t1sk3/go-utils/types@none` to avoid “ambiguous import” errors.


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

Operations that change the element type are methods too (Go 1.27 generic methods), so
pipelines read left to right:

```go
names := stream.FromSlice(users).Filter(User.IsActive).Map(User.Name).ToSlice()
byTeam := stream.FromSlice(users).GroupBy(func(u User) string { return u.Team })
lengths := stream.Of("go", "rust").Fold(0, func(n int, w string) int { return n + len(w) })
```

Operations that need `comparable`, `cmp.Ordered` or a number type are package
functions, because a method can't add a constraint to the stream's own element type. `Enumerate`, `Chunk`, `Window` and `Zip` are functions only
too: the compiler rejects their method forms as an instantiation cycle.

```go
total := stream.Sum(stream.FromSlice(orders).Map(Order.Total))
csv := stream.Join(stream.Distinct(stream.Of("a", "b", "a")), ",") // "a,b"
```

When a method's type argument can't be inferred, as with a bare `nil`, give it
explicitly: `s.Map[string](nil)`.

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
avg := stream.Window(prices, 7).Map(mean)                        // 7-day moving average
balance := changes.Scan(0, func(t, c int) int { return t + c })
```

Collectors reduce a stream and nest, so a grouping can reduce each group instead of
keeping its elements. They are the counterparts of Java's `Collectors`:

```go
averageAge := users.Collect(stream.GroupingBy(User.Team, stream.Averaging(User.Age)))
activeByTeam := users.Collect(stream.GroupingBy(User.Team,
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
pages := urls.MapConcurrent(8, fetch).ToSlice()
ok := stream.FromSlice(images).Parallel(runtime.GOMAXPROCS(0)).
	Map(resize).
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
| **Intermediate methods** | `Filter`, `Map`, `FlatMap`, `DistinctBy`, `Scan`, `MapConcurrent`, `Peek`, `Limit`, `Skip`, `TakeWhile`, `DropWhile`, `Sorted`, `Parallel` |
| **Terminal methods** | `ToSlice`, `ForEach`, `Count`, `Reduce`, `Fold`, `Collect`, `Min`, `Max`, `FindFirst`, `FindLast`, `AnyMatch`, `AllMatch`, `NoneMatch`, `Partition`, `GroupBy`, `CountBy`, `ToMap`, `ToMapMerge`, `All` |
| **Functions** | `Zip`, `Enumerate`, `Chunk`, `Window`, `Distinct`, `Sort`, `Sum`, `Average`, `Summarize`, `Min`, `Max`, `ToSet`, `ToSeq2`, `Join`, `Identity` |
| **Comparators** | `Comparing`, `Reversed`, `ThenComparing`, `NilsFirst`, `NilsLast` |
| **Predicates** | `IsEven`, `IsOdd`, `IsPositive`, `IsNegative`, `IsZero`, `Equal`, `GreaterThan`, `LessThan`, `AtLeast`, `AtMost`, `Between`, `In`, `Not`, `And`, `Or` |
| **Collectors** | `Collect`, `NewCollector`, `IntoSlice`, `IntoSet`, `Joining`, `Counting`, `Summing`, `Averaging`, `MinBy`, `MaxBy`, `Folding`, `GroupingBy`, `PartitioningBy`, `Mapping`, `FlatMapping`, `Filtering`, `CollectingAndThen`, `Teeing` |

## optional

```go
import "github.com/t1sk3/go-utils/types/optional"

port := optional.OfOk(os.LookupEnv("PORT")).OrElse("8080")

if u, ok := findUser(id).Value(); ok { ... }

name := findUser(id).Map(User.Name).OrElse("anonymous")
email := findUser(id).FlatMap(User.Email) // when User.Email returns an Optional

u, err := findUser(id).OrElseErr(ErrNotFound)
timeout := optional.FromMap(settings, "timeout").OrElse(30)
```

`Map` and `FlatMap` are methods and can change the type, so calls chain.

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

admins := stream.ToSet(stream.FromSlice(users).Filter(User.IsAdmin).Map(User.ID))
missing := required.Difference(granted)
```

`Set` has `Union`, `Intersection`, `Difference`, `SymmetricDifference`,
`IsSubsetOf`, `IsSupersetOf`, `IsDisjoint`, `Equal`, iterates with `All()` and marshals
to/from a JSON array. `Add`, `Remove`, `RemoveIf` and `RetainAll` return how many values
they changed.
Like a map, copies share storage; use `Clone` for an independent copy.

## vector

```go
import "github.com/t1sk3/go-utils/types/vector"

pos := vector.Of2(1.0, 2.0)                    // Vec2[float64]{X: 1, Y: 2}
pos = pos.Add(vel.Scale(dt))
normal := a.Sub(b).Cross(c.Sub(b)).Normalize() // Vec3
features := vector.Of(0.2, 0.7, 0.1, 0.9)     // VecN[float64], any dimension
similarity := math.Cos(features.AngleTo(query))
```

`Vec2` and `Vec3` are comparable structs that never allocate, so they work as map keys;
`VecN` is a slice, so it indexes, ranges and works with the `slices` package. All three
have `Add`, `Sub`, `Mul`, `Div`, `Scale`, `DivScalar`, `Neg`, `Abs`, `Min`, `Max`, `Clamp`,
`Sum`, `Dot`, `Length`, `LengthSquared`, `Distance`, `DistanceSquared`, `Normalize`, `Lerp`,
`AngleTo`, `Project`, `Reflect`, `IsZero` and `ApproxEqual`. `Vec3` adds `Cross` and `XY`;
`Vec2` adds a scalar `Cross`, `Perp`, `Rotate`, `Angle` and `FromPolar`.

Elements can be any integer or float type. Arithmetic stays in `T`; lengths and angles are
`float64`, and results such as `Normalize` convert back to `T`, so for integer vectors
convert first with `Convert`, `Convert2` or `Convert3`. Combining `VecN`s of different
dimensions panics.

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
