# AGENTS.md

aifactory is a Go module; see [README.md](README.md) for what it does, how to use it and how to run `make check`.

## Testing

- Tests are unit tests only. No integration tests: no real filesystem, no temp dirs (`t.TempDir`), no subprocesses (`exec.Command`), no network and no `httptest` servers.
- Collaborators are replaced by spy mocks from `go-specs`: hand-written typed adapters over `mock.Controller` (package `github.com/getsyntegrity/go-specs/mock`). No generated mocks and no other mocking library.
- Every test, new or existing, is written with the `go-specs` DSL (`specs.Describe` / `It`), with assertions through `ctx.Expect(...)`.
- `go-specs` (`github.com/getsyntegrity/go-specs`) is pinned to an exact version in `go.mod`. Upgrading it is a deliberate change, made after reading its changelog.
- `internal/jobs/jobs_test.go` is the reference example to copy.

### Pending migration

These test files still use plain `testing` with `t.Errorf` / `t.Fatal`, and some of them touch the real filesystem or run subprocesses. Do not copy their style; they will be migrated to the rule above:

- `cmd/aifactory/main_test.go`
- `internal/mcp/server_test.go`
- `internal/state/odd_test.go`
- `internal/state/state_test.go`
- `internal/store/store_test.go`
- `internal/store/artifacts_test.go`
