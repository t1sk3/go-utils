# go-utils

[![Go Reference](https://pkg.go.dev/badge/github.com/t1sk3/go-utils.svg)](https://pkg.go.dev/github.com/t1sk3/go-utils)
[![CI](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml/badge.svg)](https://github.com/t1sk3/go-utils/actions/workflows/ci.yml)

Small, dependency-free generic utilities.

> **Requires Go 1.27 or later.** The API is built on generic methods, which Go gained in
> 1.27, so that pipelines such as `s.Filter(f).Map(g)` can change the element type and
> still read left to right. That is a deliberate choice: there is no fallback for older
> toolchains.

| Package | What it gives you |
|---|---|
| [`stream`](stream) | Lazy, chainable pipelines over slices, iterators and generators |
| [`future`](future) | `Future[T]`, the typed result of work running on another goroutine, with `All`, `Any`, `Then`, `Promise` and `Pool` |
| [`concurrent`](concurrent) | `Map[K, V]` with atomic `ComputeIfAbsent`, `Compute` and `Merge`, plus `ShardedMap` and `Set` |
| [`optional`](types/optional) | A zero-allocation container for a value that may be absent |
| [`deque`](types/deque) | `Deque[T]`, a double-ended queue that also works as a FIFO queue or a stack |
| [`pq`](types/pq) | `Queue[T]`, a priority queue ordered by a comparator |
| [`pair`](types/pair) | `Pair[A, B]`, two values of possibly different types |
| [`sorted`](types/sorted) | `Map[K, V]` and `Set[T]`, kept in comparator order with floor, ceiling and range lookups |
| [`set`](types/set) | `Set[T]`, an unordered collection of distinct values |
| [`linked`](types/linked) | `Map[K, V]` and `Set[T]` that remember insertion order, which also makes an LRU cache simple |
| [`vector`](types/vector) | `VecN[T]` of any dimension, plus `Vec2[T]` and `Vec3[T]` with `X`, `Y`, `Z` fields |
| [`result`](types/result) | `Result[T]`, a success or an error as one value |
| [`triple`](types/triple) | `Triple[A, B, C]`, three values of possibly different types |
| [`multimap`](types/multimap) | `Multimap[K, V]`, any number of values per key |
| [`bimap`](types/bimap) | `BiMap[K, V]`, a one-to-one map with lookups from both sides |
| [`ring`](types/ring) | `Ring[T]`, a fixed-size buffer that overwrites its oldest value |
| [`trie`](types/trie) | `Trie[V]`, string keys with prefix search and longest-prefix match |
| [`retry`](retry) | `Do`, which calls a function again when it fails, with backoff and jitter |
| [`ratelimit`](ratelimit) | `Limiter`, a token bucket with `Allow` and `Wait` |
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
Upgrading from an older version? [CHANGELOG.md](CHANGELOG.md) lists what changed in each
release and how to adapt. To report a security problem, see [SECURITY.md](SECURITY.md).

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
functions, because a method can't add a constraint to the stream's own element type.

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
for i, name := range stream.ToSeq2(names.Enumerate()) { ... }
byID := maps.Collect(stream.ToSeq2(ids.Zip(users)))
```

Batching, sliding windows and running totals work on infinite streams too:

```go
for batch := range rows.Chunk(500).All() { db.InsertMany(batch) }
avg := prices.Window(7).Map(mean)                                // 7-day moving average
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

Two options change how the parallel stages behave. `WithContext` cuts the stream short
when a context ends, and `Unordered` lets results come out as they finish instead of in
source order, so a slow element does not hold back the rest:

```go
pages := urls.Parallel(8).WithContext(ctx).Map(fetch).ToSlice() // check ctx.Err() afterwards
urls.Parallel(8).Unordered().Map(fetch).Sequential().ForEach(index)
```

A step that can fail goes through `TryMap`, which produces a `result.Result` per element,
and `CollectResults` stops at the first error:

```go
numbers, err := stream.CollectResults(lines.TryMap(strconv.Atoi))
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
| **Sources** | `Of`, `FromSlice`, `FromSeq`, `FromSeq2`, `FromMap`, `FromOptional`, `OfNullable`, `FromChannel`, `Range`, `RangeClosed`, `Iterate`, `IterateWhile`, `Generate`, `Concat`, `Empty` |
| **Intermediate methods** | `Filter`, `Map`, `TryMap`, `FlatMap`, `DistinctBy`, `Scan`, `Zip`, `ZipWith`, `Enumerate`, `Chunk`, `Window`, `MapConcurrent`, `Peek`, `Limit`, `Skip`, `TakeLast`, `TakeWhile`, `DropWhile`, `Sorted`, `Top`, `Reverse`, `Intersperse`, `Cycle`, `Parallel` |
| **Terminal methods** | `ToSlice`, `ForEach`, `Count`, `Reduce`, `Fold`, `Collect`, `Min`, `Max`, `MinBy`, `MaxBy`, `FindFirst`, `FindLast`, `AnyMatch`, `AllMatch`, `NoneMatch`, `Partition`, `GroupBy`, `CountBy`, `ToMap`, `ToMapMerge`, `ToChannel`, `All` |
| **Functions** | `Distinct`, `Sort`, `Sum`, `Average`, `Summarize`, `Min`, `Max`, `ToSet`, `ToSeq2`, `Join`, `Flatten`, `Unzip`, `CollectResults`, `Identity` |
| **Parallel** | `Parallel`, `MapConcurrent`, `WithContext`, `Unordered`, `Filter`, `Map`, `Peek`, `ForEach`, `Sequential` |
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

`optional.Field[T]` adds a third state for JSON: left out, `null`, or a value. That is
what a PATCH request needs, where leaving a field out means "keep it" and `null` means
"clear it":

```go
type UpdateUser struct {
	Email optional.Field[string] `json:"email,omitzero"`
}
// req.Email.IsAbsent(), req.Email.IsNull(), or req.Email.Value()
```

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
to/from a JSON array, with the elements in a stable order. `Add`, `Remove`, `RemoveIf` and `RetainAll` return how many values
they changed, `AddSeq` adds from an iterator, `HasAll` and `HasAny` test several values
at once, and `Pop` takes some value out.
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
`Slerp`, `ClampLength`, `Floor`, `Ceil`, `Round`, `AngleTo`, `Project`, `Reflect`, `IsZero`
and `ApproxEqual`, and so has `Vec4[T]`. `Vec3` adds `Cross`, `XY` and `Rotate` around an
axis; `Vec2` adds a scalar `Cross`, `Perp`, `Rotate`, `Angle` and `FromPolar`; `VecN` adds
`AddInPlace`, `SubInPlace` and `ScaleInPlace` for loops that must not allocate.

Elements can be any integer or float type. Arithmetic stays in `T`; lengths and angles are
`float64`, and results such as `Normalize` convert back to `T`, so for integer vectors
convert first with `Convert`, `Convert2` or `Convert3`. Combining `VecN`s of different
dimensions panics.

## future

```go
import "github.com/t1sk3/go-utils/future"

user := future.Go(func() (User, error) { return fetchUser(ctx, id) })
orders := future.Go(func() ([]Order, error) { return fetchOrders(ctx, id) })

u, err := user.Get(ctx)
o, err := orders.Get(ctx)
```

`Go` runs a function on a new goroutine and returns a `Future[T]` of its result, which
can be read any number of times from any goroutine. `Then` chains a second step that may
change the type, `All` waits for every future and fails fast, and `Any` returns the first
success:

```go
total := future.Go(fetchCart).Then(priceCart)
pages, err := future.All(ctx, future.Go(fetchA), future.Go(fetchB))
fastest, err := future.Any(ctx, future.Go(fromMirror1), future.Go(fromMirror2))
```

A panic in the function becomes a `*future.PanicError` instead of crashing the program;
its `Stack` field holds the trace. Canceling the context passed to `Get` stops the wait,
not the work: to cancel the work, start it with `GoCtx` and have the function watch the
context it is given.

`AllSettled` waits for every future and returns each outcome as a `result.Result`,
`Recover` and `OrElse` substitute a value for a failure, and `TryGet` and `IsDone` look
without waiting. A `Promise` is a Future completed by hand, for callback-style code, and
a `Pool` limits how many functions run at once:

```go
pool := future.NewPool(8)
for _, url := range urls {
	pages = append(pages, pool.Go(func() (Page, error) { return fetch(url) }))
}
results, err := future.All(ctx, pages...)
```

## concurrent

```go
import "github.com/t1sk3/go-utils/concurrent"

var cache concurrent.Map[string, *Template] // the zero value is ready to use

tmpl := cache.ComputeIfAbsent(name, parse) // parse runs once per missing key
hits.Merge(path, 1, func(a, b int) int { return a + b })
```

`Map[K, V]` is safe for concurrent use and makes compound updates atomic:
`ComputeIfAbsent`, `Compute`, `Merge` and `PutIfAbsent`, next to `Get`, `Has`, `Put`,
`Remove`, `Len`, `Clear`, `Snapshot` and `All`. One read-write lock guards the whole map,
so prefer `sync.Map` for read-mostly maps that need none of the compound updates.
Functions passed to the Map run while it is locked and must not call back into it.

`Map` also has `GetOrDefault`, `Replace`, `ComputeIfPresent`, `RemoveIf`, `Keys` and
`Values`, and the functions `CompareAndSwap` and `CompareAndDelete` for comparable values.
`ShardedMap` offers the same per-key operations over independently locked parts, which
scales across cores at the price of whole-map operations no longer being atomic. `Set`
is a concurrent set whose `Add` reports whether the value was new, so exactly one
goroutine can claim it.

## deque

```go
import "github.com/t1sk3/go-utils/types/deque"

var q deque.Deque[Job] // the zero value is ready to use
q.PushBack(a, b)
next, ok := q.PopFront().Value() // FIFO; use PopBack for a stack
```

A ring buffer with amortized constant-time `PushFront`, `PushBack`, `PopFront` and
`PopBack`, plus `Front`, `Back`, `At`, `Set`, `Rotate`, `All` and `Backward`. The buffer
grows and shrinks with its contents, and it marshals to/from a JSON array.

## pq

```go
import "github.com/t1sk3/go-utils/types/pq"

tasks := pq.New(func(a, b Task) int { return cmp.Compare(a.Deadline, b.Deadline) })
tasks.Push(t1, t2)
next, ok := tasks.Pop().Value() // the earliest deadline
```

A binary heap ordered by a comparator, so `cmp.Compare` gives a min-heap and
`stream.Reversed` of it a max-heap; `pq.NewMin` and `pq.NewMax` are the shorthand for
ordered types. `Drain` iterates in priority order, `PushPop` does a push and a pop in one
pass, and `RemoveFunc` removes by predicate.

## sorted

```go
import "github.com/t1sk3/go-utils/types/sorted"

prices := sorted.NewOrderedMap[time.Time, float64]()
prices.Put(t, 9.99)
latest := prices.Floor(now) // the most recent price at or before now
for at, price := range prices.Range(start, end) { ... }
```

`Map[K, V]` and `Set[T]` keep their contents in comparator order in an AVL tree. Besides
lookups in logarithmic time they have `First`, `Last`, `PopFirst`, `PopLast`, `Floor`,
`Ceiling`, `Lower`, `Higher` and `Backward`. Ranges come half-open (`Range`), inclusive
(`RangeClosed`), open-ended (`From`, `Until`) and descending (`BackwardFrom`,
`BackwardRange`). `Rank` gives a key's position and `At` the entry at a position, both in
logarithmic time, so `m.At(m.Len() / 2)` is the median.

## linked

```go
import "github.com/t1sk3/go-utils/types/linked"

cache := linked.NewMap[string, Page]()
if page, ok := cache.Get(key).Value(); ok {
	cache.MoveToBack(key) // most recently used
	return page
}
cache.Put(key, load(key))
if cache.Len() > limit {
	cache.PopFirst() // least recently used
}
```

`Map[K, V]` and `Set[T]` are hash-based and iterate in insertion order. `MoveToBack`,
`MoveToFront`, `PopFirst` and `PopLast` take constant time, which is what an LRU cache
needs. `linked.NewLRU` is that cache ready-made, with a capacity and an eviction
callback. Both types marshal to and from JSON in order, which a built-in map cannot do.

## pair

`pair.Pair[A, B]` holds two values. It is what `Zip`, `Enumerate` and `FromMap` produce,
and `First`, `Last` and friends on the maps return. `p.Unpack()` gives both values back.

## result and triple

```go
import "github.com/t1sk3/go-utils/types/result"

r := result.Of(strconv.Atoi(s)) // a success or an error, as one value
n := r.Map(double).OrElse(0)
v, err := r.Get() // back to Go's two results
```

`Result[T]` is for the places that only have room for one value: a slice of outcomes, a
channel, a stream. Ordinary functions should keep returning `(T, error)`.
`triple.Triple[A, B, C]` is `Pair` with a third value.

## multimap, bimap, ring and trie

```go
var byTag multimap.Multimap[string, Post] // any number of values per key
byTag.Put("go", post1, post2)

var codes bimap.BiMap[string, int] // one-to-one, looked up from either side
codes.Put("NL", 31)
country := codes.GetKey(31)

recent := ring.New[string](100) // keeps the last 100, overwriting the oldest
recent.Push(line)

var routes trie.Trie[Handler] // string keys, searched by prefix
routes.Put("/api/users/", usersHandler)
match := routes.LongestPrefix("/api/users/42")
for path := range routes.WithPrefix("/api/") { ... }
```

## retry and ratelimit

```go
import "github.com/t1sk3/go-utils/retry"

page, err := retry.Do(ctx, retry.Policy{Attempts: 5, Delay: 100 * time.Millisecond, Multiplier: 2, Jitter: 0.2},
	func(ctx context.Context) (Page, error) { return fetch(ctx, url) })
```

`Do` calls the function until it succeeds, the attempts run out or the context ends.
Wrap an error in `retry.Permanent`, or set `Policy.RetryIf`, to give up at once on
failures that will not pass.

```go
import "github.com/t1sk3/go-utils/ratelimit"

limiter := ratelimit.New(10, 20) // 10 per second, bursts of up to 20
if err := limiter.Wait(ctx); err != nil { return err }
```

`Limiter` is a token bucket that is safe for concurrent use. `Wait` blocks until an event
may happen and serves waiting callers in order; `Allow` never waits, for dropping what is
over the limit.

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
