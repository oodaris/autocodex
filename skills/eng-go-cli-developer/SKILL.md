---
name: eng-go-cli-developer
description: Build reliable Go CLI commands with stable outputs and tests.
version: 0.1.0
---

# Go CLI Developer

## Repo anchors (Autocodex)
- CLI_PATH: `cmd/autorunner/`
- INTERNAL_PATH: `internal/`
- TEST_COMMANDS
  - `go test ./...`
  - `go vet ./...`
  - `gofmt -w $(rg --files -g '*.go')`

## When to use
- Adding or modifying CLI commands or flags.

## Preconditions
- CLI goals and expected workflows are defined.
- If contracts must change, STOP and update contracts first.

## Inputs to confirm
- Command structure and flags
- Expected outputs (human vs JSON)
- Exit code semantics

## Required artifacts
- Deterministic CLI behavior
- Help text or examples for new commands
- Tests for parsing or core flows

## Quick path
- Define flags and output format.
- Keep CLI thin; move logic to packages.
- Add tests.

## Steps
1) Implement CLI parsing.
2) Delegate to internal packages.
3) Ensure stdout is for output, stderr for logs/errors.
4) Add tests for parsing/output.

## Failure modes and responses
- **Unclear UX**: ask for expected CLI workflow.
- **Unstable output**: add JSON output and tests.

## Definition of done
- CLI is deterministic, documented, and tested.

## Example (minimal)
- **Command**: `autorunner plugins --action list`
- **Output**: JSON list of plugins.
