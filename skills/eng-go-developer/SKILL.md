---
name: eng-go-developer
description: Implement Go code with clear errors, tests, and observability.
version: 0.1.0
---

# Go Developer

## Repo anchors (Autocodex)
- INTERNAL_PATH: `internal/`
- TEST_COMMANDS
  - `go test ./...`
  - `go vet ./...`
  - `gofmt -w $(rg --files -g '*.go')`

## When to use
- Implementing or refactoring Go logic, services, or libraries.

## Preconditions
- Scope and expected behavior are defined.
- If contracts must change, STOP and update contracts first.

## Inputs to confirm
- Target package and entry points
- Concurrency expectations
- Required tests

## Required artifacts
- Go code with explicit errors
- Tests for happy path and edge cases
- Observability hooks for user-visible failures

## Quick path
- Implement small, composable functions.
- Add tests and run Go checks.

## Steps
1) Implement with explicit errors and context propagation.
2) Avoid global state; inject dependencies.
3) Add tests.

## Failure modes and responses
- **Missing contracts**: stop and update specs.
- **No tests**: block until tests are added.

## Definition of done
- Code is correct, tested, and observable.

## Example (minimal)
- **Change**: add a new internal package.
- **Checks**: gofmt + go test + go vet.
