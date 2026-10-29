// Package types is the root of a small, dependency-free module holding the value
// types and type constraints shared by the rest of go-utils. It lives in its own
// module so that libraries can accept or return these types without pulling in
// any of the utilities built on top of them.
//
// The types themselves live in subpackages:
//
//   - [github.com/t1sk3/go-utils/types/optional]: Optional, a value that may be absent
//   - [github.com/t1sk3/go-utils/types/pair]: Pair, two values of possibly different types
//   - [github.com/t1sk3/go-utils/types/set]: Set, an unordered collection of distinct values
//   - [github.com/t1sk3/go-utils/types/constraints]: Integer, Number and friends for generic code
package types
