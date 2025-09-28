# Repository Guidelines

## Project Structure & Module Organization
- Module: `github.com/xyhelper/xyqueue` (Go module managed by `go.mod`).
- Source lives in Go packages at the repo root (e.g., `queue/`) or under `internal/` for private APIs; examples may go in `examples/`.
- Tests are colocated with code using `_test.go` files (e.g., `queue/queue_test.go`).
- Keep packages small and purpose‑focused; avoid circular deps. Package names are short, lowercase, no underscores.

## Build, Test, and Development Commands
- Init/refresh deps: `go mod tidy` — syncs `go.mod`/`go.sum`.
- Format: `go fmt ./...` — formats all Go files.
- Lint (basic): `go vet ./...` — static checks for common issues.
- Build library/packages: `go build ./...` — ensures code compiles across packages.
- Run tests: `go test -v ./...` — verbose test run for all packages.
- Race checks: `go test -race ./...` — detects data races (useful for concurrent code).
- Coverage report: `go test -coverprofile=cover.out ./... && go tool cover -func=cover.out`.

## Coding Style & Naming Conventions
- Use `go fmt` (no custom indentation rules). Prefer idiomatic Go: short names for locals, exported identifiers in `CamelCase` with doc comments.
- Package names: lowercase, singular, no underscores (e.g., `queue`). File names use underscores only for suffixes like `_test.go`.
- Errors: wrap with `%w` (`fmt.Errorf`) and compare via `errors.Is/As`.
- Comments: keep package and exported symbol docs up to date; include usage examples where helpful.

## Testing Guidelines
- Framework: standard library `testing`. Prefer table‑driven tests and `t.Run` subtests.
- Naming: `<file>_test.go`, test funcs `TestXxx`, benchmarks `BenchmarkXxx`, examples `ExampleXxx`.
- Aim for clear coverage of FIFO and de‑dup behaviors; include edge cases and concurrency where applicable.
- Run locally: `go test -race -v ./...` before pushing.

## Commit & Pull Request Guidelines
- Commits: small, focused, imperative mood (e.g., "queue: avoid duplicate enqueue"). Reference issues (`#123`) when applicable.
- PRs: include summary, motivation, and scope; link issues; add tests for new behavior; include before/after notes or micro‑benchmarks for performance‑sensitive changes.
- Ensure CI‑green locally (format, vet, test) before requesting review.
