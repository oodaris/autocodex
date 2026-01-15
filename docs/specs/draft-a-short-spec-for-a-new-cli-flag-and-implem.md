# CLI Status Latest Flag

## Metadata
```yaml
id: cli-status-latest-flag
owner: cli
status: draft
created: 2026-01-15
updated: 2026-01-15
```

## Problem statement
The `autocodex status` command lists every run, which makes it cumbersome to quickly check the most recent run in active repos. Users want a single, concise status for the latest run without scrolling or piping output. Assumption: run ID ordering reflects recency and is acceptable for "latest" semantics.

## Goals
- Add a `--latest` flag to `autocodex status` that returns only the most recent run's status.
- Preserve existing `autocodex status` behavior when the flag is not provided.
- Allow `--latest` to work with `--json` output.

## Non-goals
- Changing how runs are ordered or stored.
- Adding new status fields or changing the status output format.
- Modifying API or UI behavior.

## Requirements
### Functional
- `autocodex status --latest` returns a single status entry for the most recent run (by run ID sort order).
- `--latest` supports `--json` output.
- `--latest` cannot be used with `--run` and should error with a clear message.
- If no runs exist, the command prints "No runs found" and exits successfully.
- README documents the new flag.

### Non-functional
- No new dependencies or config changes.
- Maintain current output format and performance characteristics.

## Interfaces / data contracts
- CLI flag: `autocodex status --latest` (no schema or API changes).

## Acceptance criteria
- New unit tests cover latest selection and `--latest` + `--run` conflict.
- `go test ./cmd/autocodex` passes.
- README includes an example using `--latest`.

## Open questions
- Should "latest" be based on `StartedAt` timestamps instead of run ID ordering?
- Should a short flag alias (e.g., `-l`) be added later?

## Risks
- Run ID sorting may not perfectly match true recency if multiple runs start within the same second; this mirrors existing `latest` behavior elsewhere.
- Users may expect `--latest` to filter by status (e.g., running only), which is out of scope.

## References
- cmd/autocodex/main.go
- cmd/autocodex/main_test.go
- internal/state/state.go
- README.md
