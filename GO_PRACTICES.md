# Go Engineering Standards

This document outlines the preferred Go coding practices for this repository and the enforcement steps expected in development and CI.

## Core Principles
- Favor readability and explicitness over cleverness.
- Keep packages small and cohesive; avoid cyclic dependencies.
- Prefer composition over inheritance-like patterns; use interfaces to describe behavior, not data.
- Design APIs to be easy to use correctly and hard to misuse.

## Style and Formatting
- Run `go fmt ./...` on every change. Do not commit unformatted files.
- Order imports using the standard three-block grouping: stdlib, third-party, local modules.
- Keep files focused; avoid long functions—extract helpers when a function exceeds ~50 lines of logic.
- Name errors `ErrSomething`, interfaces with behavior-oriented names (e.g., `Reader`, `Fetcher`), and avoid stutter (`storage.Store` not `storage.Storage`).

## Error Handling
- Return contextual errors using `%w` for wrapping when the caller may need to inspect the cause.
- Handle errors explicitly; avoid ignoring `err` unless it is safe and justified with a short comment.
- Distinguish between expected control-flow errors and unexpected failures; prefer sentinel errors or typed errors over string comparisons.
- Avoid panics in library code; reserve panics for truly unrecoverable situations (e.g., programmer errors).

## Concurrency and Resource Management
- Use `context.Context` for request-scoped work; accept `context.Context` as the first parameter for functions that perform I/O or long-running work.
- Prefer channels and `sync.WaitGroup` over manual goroutine tracking; avoid goroutine leaks by ensuring exit conditions and cancellation.
- Guard shared state with mutexes or channel ownership; avoid race conditions by designing clear ownership of data.
- Use `time.Ticker` and `time.Timer` carefully—stop and drain them to prevent leaks.

## Testing
- Write table-driven tests and favor subtests for clarity and coverage of edge cases.
- Use `t.Helper()` in helper functions to preserve accurate failure line numbers.
- Keep tests deterministic; avoid reliance on real time or network where possible by injecting clocks and dependencies.
- Benchmark performance-sensitive code with `go test -bench=.` and document expectations.

## Logging and Observability
- Log actionable information with structured fields; avoid excessive log volume or sensitive data.
- Prefer returning errors over logging inside libraries; let callers decide how to surface the issue.
- Use metrics/tracing hooks where available to surface latency, error rates, and key business signals.

## Dependencies and Modules
- Prefer standard library packages when feasible.
- Pin versions in `go.mod`; avoid replacing modules with local paths in committed code.
- Remove unused dependencies regularly with `go mod tidy`.
- Vet third-party dependencies for license compatibility and security posture.

## Security and Validation
- Validate all external inputs; avoid trusting JSON/XML or environment data without checks.
- Use constant-time comparisons for secrets (`subtle.ConstantTimeCompare`).
- Avoid embedding secrets in code; load configuration via environment variables or secret managers.
- Use `govulncheck` to detect known vulnerabilities before releases.

## Performance and Footprint
- Choose data structures deliberately; benchmark before optimizing.
- Avoid premature allocation—reuse buffers with `sync.Pool` only when profiling proves benefit.
- Protect critical paths from unnecessary conversions and reflection.

## Required Enforcement Steps
Run these locally and in CI for every change:
1. `go fmt ./...` for formatting.
2. `go vet ./...` to catch common mistakes.
3. `staticcheck ./...` (or `golangci-lint run`) for extended linting; configure linters per project needs.
4. `go test ./...` with `-race` where feasible for concurrency safety.
5. `govulncheck ./...` before releases to catch known vulnerabilities.

## Pull Request Checklist
- Tests and linters pass; include command outputs in the PR description when possible.
- New code paths are covered with tests or include justification when not feasible.
- Public-facing APIs include doc comments; configuration and behavior changes are documented in README or relevant guides.

Following these standards will keep the codebase predictable, secure, and maintainable as it grows.
