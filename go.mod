module github.com/t1sk3/go-utils

go 1.24.8

require github.com/t1sk3/go-utils/types v0.0.0

// Resolve the types module from this repository so both modules can be changed in
// one commit. Remove this and require a tagged types/vX.Y.Z before tagging a release:
// replace directives are ignored by modules that depend on this one.
replace github.com/t1sk3/go-utils/types => ./types
