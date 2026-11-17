---
name: go-unit-testing
description: Rules for writing or changing Go tests in this repo.
---

## Testing

- Tests are unit tests only. No integration tests: no real filesystem, no temp dirs (`t.TempDir`), no subprocesses (`exec.Command`), no network and no `httptest` servers.
- Collaborators are replaced by spy mocks from `go-specs`: hand-written typed adapters over `mock.Controller` (package `github.com/getsyntegrity/go-specs/mock`). No generated mocks and no other mocking library.
- Every test, new or existing, is written with the `go-specs` DSL (`specs.Describe` / `It`), with assertions through `ctx.Expect(...)`.
- `Describe` names represent the behaviour under test: each one explains the behaviour being tested, not the function, type or file that implements it. For example, `"creating a job from a request ID"` in `internal/jobs/jobs_test.go`.
- `go-specs` (`github.com/getsyntegrity/go-specs`) is pinned to an exact version in `go.mod`. Upgrading it is a deliberate change, made after reading its changelog.
- `internal/jobs/jobs_test.go` is the reference example to copy.

### Pending migration

These test files still use plain `testing` with `t.Errorf` / `t.Fatal`, and some of them touch the real filesystem or run subprocesses. Do not copy their style; they will be migrated to the rule above:

- `internal/store/store_test.go`
- `internal/store/artifacts_test.go`
