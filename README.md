# go-utils

Small, dependency-free generic utilities for Go 1.24+.

The repository holds two modules:

| Module | Package | What it gives you |
|---|---|---|
| `github.com/t1sk3/go-utils` | [`stream`](stream) | Lazy, chainable pipelines over slices, iterators and generators |
| `github.com/t1sk3/go-utils/types` | [`optional`](types/optional) | A zero-allocation container for a value that may be absent |
| | [`pair`](types/pair) | `Pair[A, B]`, two values of possibly different types |
| | [`set`](types/set) | `Set[T]`, an unordered collection of distinct values |
| | [`constraints`](types/constraints) | `Integer`, `Number`, `Float`, ... constraints for generic code |

`types` holds the shared vocabulary and has no dependencies, so your own libraries
can accept or return an `optional.Optional` without pulling in `stream` or anything
else built on top of it:

```bash
go get github.com/t1sk3/go-utils/types   # just the types
go get github.com/t1sk3/go-utils         # the utilities (brings in types)
```

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

| | |
|---|---|
| **Sources** | `Of`, `FromSlice`, `FromSeq`, `FromSeq2`, `FromMap`, `Range`, `Iterate`, `Generate`, `Concat`, `Empty` |
| **Intermediate methods** | `Filter`, `Map` (same type), `Peek`, `Limit`, `Skip`, `TakeWhile`, `DropWhile`, `Sorted` |
| **Terminal methods** | `ToSlice`, `ForEach`, `Count`, `Reduce`, `Min`, `Max`, `FindFirst`, `AnyMatch`, `AllMatch`, `NoneMatch`, `Partition`, `All` |
| **Generic functions** | `Map`, `FlatMap`, `Zip`, `Enumerate`, `Chunk`, `Window`, `Scan`, `Distinct`, `DistinctBy`, `Sort`, `Fold`, `Sum`, `Min`, `Max`, `GroupBy`, `ToMap`, `ToSet`, `ToSeq2`, `Join` |

## optional

```go
import "github.com/t1sk3/go-utils/types/optional"

port := optional.OfOk(os.LookupEnv("PORT")).OrElse("8080")

if u, ok := findUser(id).Value(); ok { ... }

name := optional.Map(findUser(id), func(u User) string { return u.Name }).OrElse("anonymous")
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
`IsSubsetOf`, `Equal`, iterates with `All()` and marshals to/from a JSON array.
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
