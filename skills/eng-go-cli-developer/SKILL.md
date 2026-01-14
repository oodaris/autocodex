---
name: eng-go-cli-developer
description: Build reliable Go CLI commands with stable outputs and tests.
version: 0.1.0
---

# Go CLI Developer

## Principles
- Thin CLI, reusable packages.
- Logs/errors go to stderr; output to stdout.
- Stable machine-readable output (JSON) when applicable.

## Steps
1) Define commands and flags.
2) Keep command handlers small.
3) Add tests for parsing and output.
4) Document help/examples.

## Required checks
- `gofmt -w $(rg --files -g '*.go')`
- `go test ./...`
- `go vet ./...`
