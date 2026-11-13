# Changelog

Notable changes to this module. Versions follow [semantic versioning](https://semver.org);
the module is at v0, so minor releases may break the API.

## Unreleased

### Fixed

- `concurrent.Map.ComputeIfAbsent` no longer leaves the Map read-locked when the lookup
  panics on an unhashable key. Every later write used to deadlock.
- `future.All` and `future.Any` panic on the caller's goroutine for a nil `*Future`,
  instead of crashing the program from a goroutine the caller cannot guard.
- `future.All` and `future.Any` return an outcome that is already available even if the
  context has ended, as `Get` does. `All` used to return the context's error.
- `stream`: `Distinct`, `DistinctBy`, `ToMap`, `ToMapMerge`, `Chunk` and `Window` no longer
  size a buffer from the stream's size or the caller's `n`. `Distinct(Range(0, 50_000_000)).Limit(3)`
  allocated 1.1 GB and `Chunk(math.MaxInt)` panicked; buffers now start at no more than
  1024 elements and grow.
- `vector`: `Vec3` and `VecN` `Length` and `Distance` no longer overflow to `+Inf` or
  underflow to 0 for components whose length is representable, matching `Vec2`.
  `Normalize` of a very short vector now returns a unit vector.
- `set.Set.RemoveIf` removes values that are not equal to themselves, such as NaN.
- `stream.And`, `stream.Or` and `stream.Concat` copy their arguments, so changing the
  slice passed as `xs...` afterwards no longer changes or breaks the result.

### Changed

- `future.PanicError.Error` no longer includes the stack trace, which leaked source
  paths into logs and responses. Read the `Stack` field to log it.
- `set.Set.MarshalJSON` sorts the elements by their encoded form, so the output is the
  same on every run.
- `deque.Deque` halves its buffer when only a quarter is in use, instead of keeping its
  largest size forever.
- `linked.Map.Put` and `sorted.Map.Put` return the previous value as an `Optional`, as
  `concurrent.Map.Put` does. Calls that ignore the result are unaffected.

### Added

- New packages:
  - `types/result`: `Result[T]`, a success or an error as one value.
  - `types/triple`: `Triple[A, B, C]`.
  - `types/multimap`: `Multimap[K, V]`, any number of values per key.
  - `types/bimap`: `BiMap[K, V]`, a one-to-one map with lookups from both sides.
  - `types/ring`: `Ring[T]`, a fixed-size buffer that overwrites its oldest value.
  - `types/trie`: `Trie[V]`, string keys with prefix search and longest-prefix match.
  - `retry`: `Do`, with attempts, backoff, jitter, `RetryIf` and `Permanent`.
  - `ratelimit`: `Limiter`, a token bucket with `Allow` and `Wait`.
- `stream`: `Reverse`, `Intersperse`, `Cycle`, `TakeLast`, `Top`, `MinBy`, `MaxBy`,
  `ZipWith`, `TryMap`, `ToChannel`, and the functions `Unzip`, `Flatten`,
  `CollectResults` and `FromChannel`. Parallel streams gain `WithContext`, which cuts
  the stream short when a context ends, and `Unordered`, which lets results come out
  as they finish.
- `future`: `GoCtx`, `IsDone`, `TryGet`, `Recover`, `OrElse`, `AllSettled`, `Promise` and
  `Pool`.
- `concurrent`: `Map.GetOrDefault`, `Replace`, `ComputeIfPresent`, `RemoveIf`, `Keys` and
  `Values`; the functions `CompareAndSwap` and `CompareAndDelete`; `ShardedMap`; `Set`.
- `optional`: `Equal`, `EqualFunc`, and `Field[T]`, which tells a JSON field that was
  left out from one that was `null`.
- `linked`: JSON marshalling in insertion order for `Map` and `Set`, and `LRU`.
- `sorted`: `RangeClosed`, `From`, `Until`, `BackwardFrom`, `BackwardRange`, and `Rank`
  and `At` in logarithmic time, on both `Map` and `Set`.
- `pq`: `NewMin`, `NewMax`, `Collect`, `PushPop` and `RemoveFunc`.
- `set`: `AddSeq`, `Pop`, `HasAll`, `HasAny` and `ToSlice`.
- `deque`: `Set`, `Rotate` and JSON marshalling.
- `vector`: `Vec4`, `Slerp`, `ClampLength`, `Floor`, `Ceil` and `Round` on every vector
  type, `Vec3.Rotate` around an axis, and `AddInPlace`, `SubInPlace` and `ScaleInPlace`
  on `VecN`.
- `optional.Optional.OrZero`, a name for `Get` that says what happens when empty.
- `pq.Queue.String`.
- Fuzz tests for `sorted.Map`, `linked.Map`, `deque.Deque` and `pq.Queue`.
- CI runs `staticcheck`, `govulncheck` and the tests on a 32-bit platform.

## v0.6.2 and v0.6.3

The two tags are the same commit.

`Zip`, `Enumerate`, `Chunk` and `Window` are methods, and the functions are gone:

| Removed | Use instead |
|---|---|
| `stream.Zip(a, b)` | `a.Zip(b)` |
| `stream.Enumerate(s)` | `s.Enumerate()` |
| `stream.Chunk(s, n)`, `stream.Window(s, n)` | `s.Chunk(n)`, `s.Window(n)` |

The results have the same types as before (`Stream[[]T]`, `Stream[pair.Pair[...]]`).

## v0.6.1

Removes the package functions that v0.6.0 kept next to the new generic methods. Rewrite
each call with its first argument as the receiver:

| Removed | Use instead |
|---|---|
| `stream.Map(s, f)`, `stream.FlatMap`, `stream.DistinctBy`, `stream.Fold`, `stream.Scan`, `stream.GroupBy`, `stream.CountBy`, `stream.ToMap`, `stream.ToMapMerge`, `stream.Collect`, `stream.MapConcurrent` | `s.Map(f)`, `s.FlatMap(…)`, … |
| `stream.ParallelMap(p, f)` | `p.Map(f)` |
| `optional.Map(o, f)`, `optional.FlatMap(o, f)` | `o.Map(f)`, `o.FlatMap(f)` |
| `future.Then(fut, f)` | `fut.Then(f)` |
| `stream.FromArray(s)` (already deprecated) | `stream.FromSlice(s)` |

With explicit type arguments, drop the first one, which the receiver now supplies:
`stream.Map[int, string](s, nil)` becomes `s.Map[string](nil)`.

## v0.6.0

Operations that change the element type became methods, using Go 1.27 generic methods.
The minimum Go version is 1.27.

## v0.5.1

`types` is no longer a separate module. Versions up to v0.5.0 shipped it as
`github.com/t1sk3/go-utils/types`; if your `go.mod` still requires it, remove it with
`go get github.com/t1sk3/go-utils/types@none` to avoid “ambiguous import” errors.
