# Releasing

This repository holds two Go modules, and each is released by its own tag:

| Module | Directory | Tag |
|---|---|---|
| `github.com/t1sk3/go-utils` | `.` | `vX.Y.Z` |
| `github.com/t1sk3/go-utils/types` | `types/` | `types/vX.Y.Z` |

## Local development

`go.work` puts both modules in one workspace, so a change to `types` is picked up
by `stream` straight away and both can change in one commit. `go build`, `go test`
and `go vet` use the workspace; `go mod tidy` and anyone installing the module do
not, so the root `go.mod` must always require a *published* `types` version.

Never add a `replace` directive to a `go.mod`: modules that depend on this one
ignore it, so the published module would fail to build. CI rejects them.

## Cutting a release

Release `types` first whenever it changed, because the root module depends on it.

1. Merge everything into `main` and make sure CI is green.
2. If `types/` changed since its last tag, tag and push it:

   ```bash
   git tag types/v0.2.0
   git push origin types/v0.2.0
   ```

3. If the root module uses anything new from `types`, point it at that release and
   commit the change to `main`:

   ```bash
   GOWORK=off GOPROXY=direct go get github.com/t1sk3/go-utils/types@v0.2.0
   GOWORK=off go mod tidy
   ```

   (`GOPROXY=direct` skips the module proxy, which can take a few minutes to see a
   brand-new tag.)

4. Tag and push the root module:

   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

5. Ask the proxy for the new versions so they show up on pkg.go.dev straight away:

   ```bash
   GOPROXY=https://proxy.golang.org GO111MODULE=on go list -m github.com/t1sk3/go-utils/types@v0.2.0
   GOPROXY=https://proxy.golang.org GO111MODULE=on go list -m github.com/t1sk3/go-utils@v0.2.0
   ```

A pushed tag is permanent: the module proxy and checksum database keep that version
forever, even if the tag is deleted. If a release is broken, publish a new patch
version, and add a `retract` directive to `go.mod` for the bad one.

## Versioning

Both modules are at v0, so minor versions may break the API. Releasing v2 or later
of a module requires changing its module path to end in `/v2`.
