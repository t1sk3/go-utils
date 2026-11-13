// Package types groups the value types and type constraints shared by the rest of
// go-utils. Its subpackages depend on nothing outside types, so libraries can accept
// or return these types without pulling in any of the utilities built on top of them.
//
// The types themselves live in subpackages:
//
//   - [github.com/t1sk3/go-utils/types/optional]: Optional, a value that may be absent
//   - [github.com/t1sk3/go-utils/types/deque]: Deque, a double-ended queue
//   - [github.com/t1sk3/go-utils/types/pq]: Queue, a priority queue ordered by a comparator
//   - [github.com/t1sk3/go-utils/types/pair]: Pair, two values of possibly different types
//   - [github.com/t1sk3/go-utils/types/sorted]: Map and Set, kept in comparator order
//   - [github.com/t1sk3/go-utils/types/set]: Set, an unordered collection of distinct values
//   - [github.com/t1sk3/go-utils/types/linked]: Map and Set that remember insertion order
//   - [github.com/t1sk3/go-utils/types/vector]: VecN, Vec2 and Vec3, Euclidean vectors with the usual arithmetic
//   - [github.com/t1sk3/go-utils/types/result]: Result, a success or an error as one value
//   - [github.com/t1sk3/go-utils/types/triple]: Triple, three values of possibly different types
//   - [github.com/t1sk3/go-utils/types/multimap]: Multimap, any number of values per key
//   - [github.com/t1sk3/go-utils/types/bimap]: BiMap, a one-to-one map with lookups from both sides
//   - [github.com/t1sk3/go-utils/types/ring]: Ring, a fixed-size buffer that overwrites its oldest value
//   - [github.com/t1sk3/go-utils/types/trie]: Trie, string keys with prefix search
//   - [github.com/t1sk3/go-utils/types/constraints]: Integer, Number, Numeric and friends for generic code
package types
