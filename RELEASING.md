# Releasing

This repository is a single Go module, `github.com/t1sk3/go-utils`, released by
`vX.Y.Z` tags. All packages, including `types` and its subpackages, ship together in
each release.

Never add a `replace` directive to `go.mod`: modules that depend on this one ignore
it, so the published module would fail to build. CI rejects them.

Do not add a `go.mod` to a subdirectory. That turns it into a separate module with
its own `dir/vX.Y.Z` tags, which this repository moved away from in v0.5.1.

## Cutting a release

1. Rename the `Unreleased` section of [CHANGELOG.md](CHANGELOG.md) to the new version.

2. Merge everything into `main` and make sure CI is green. Locally:

   ```bash
   go mod tidy
   go build ./...
   go test ./...
   ```

   `go mod tidy` must leave `go.mod` and `go.sum` unchanged.

3. Tag and push:

   ```bash
   git tag v0.6.0
   git push origin v0.6.0
   ```

4. Ask the proxy for the new version so it shows up on pkg.go.dev straight away:

   ```bash
   GOPROXY=https://proxy.golang.org GO111MODULE=on go list -m github.com/t1sk3/go-utils@v0.6.0
   ```

   Or open `https://pkg.go.dev/github.com/t1sk3/go-utils@v0.6.0` and press **Request**.

A pushed tag is permanent: the module proxy and checksum database keep that version
forever, even if the tag is deleted or moved. Never move a tag. If a release is broken,
publish a new patch version, and add a `retract` directive to `go.mod` for the bad one.

## Versioning

The module is at v0, so minor versions may break the API. Releasing v2 or later
requires changing the module path to end in `/v2`.

## History

Up to v0.5.0, `types` was a separate module, `github.com/t1sk3/go-utils/types`,
released with `types/vX.Y.Z` tags. Those versions stay on the proxy. Users who still
require that module should remove it with `go get github.com/t1sk3/go-utils/types@none`
to avoid “ambiguous import” errors.
