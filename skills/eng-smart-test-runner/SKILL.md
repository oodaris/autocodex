---
name: eng-smart-test-runner
description: Run the most relevant tests and report failures precisely.
version: 0.1.0
---

# Smart Test Runner

## Repo anchors (Autocodex)
- TEST_COMMANDS
  - `go test ./...`
  - `go vet ./...`

## When to use
- Running tests after a change.

## Preconditions
- Code builds locally.

## Inputs to confirm
- Change scope and affected packages

## Required artifacts
- Commands run
- Failures with file/line
- Suggested next tests

## Quick path
- Run `go test ./...` then `go vet ./...`.

## Steps
1) Run relevant tests.
2) Capture failures with context.
3) Suggest next diagnostics.

## Failure modes and responses
- **Flaky tests**: record and retry once.
- **Missing test tools**: stop and report.

## Definition of done
- Tests executed and results summarized.

## Example (minimal)
- **Output**: “go test ./... passed.”
