---
name: eng-go-developer
description: Implement Go code with clear errors, tests, and observability.
version: 0.1.0
---

# Go Developer

## Principles
- Explicit errors with context.
- Deterministic behavior.
- Context propagation for timeouts/cancellation.

## Required checks
- `gofmt -w $(rg --files -g '*.go')`
- `go test ./...`
- `go vet ./...`
