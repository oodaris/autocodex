# Autonomy strictness evidence

Date: 2026-01-20

## Summary
Validated strict autonomy behaviors using the in-repo autonomy smoke + strictness tests (fake codex executor + fake bd). The tests exercise the full autonomy controller loop, required ACTIONS handling, required next-bead behavior, and bd fail-fast checks without invoking external services.

## Commands
```bash
go test ./internal/autonomy -run TestAutonomyLoopSmoke

go test ./internal/autonomy -run TestAutonomyRequiresActions
go test ./internal/autonomy -run TestAutonomyRequiresNextWhenMultipleBeadsReady
go test ./internal/autonomy -run TestAutonomyRequireBDFailsFast
```

## Results
- `TestAutonomyLoopSmoke`: passes; loop runs end-to-end with ACTIONS and marks bead done via fake bd.
- `TestAutonomyRequiresActions`: passes; missing ACTIONS causes autonomy failure.
- `TestAutonomyRequiresNextWhenMultipleBeadsReady`: passes; missing explicit next bead causes autonomy failure when multiple beads are ready.
- `TestAutonomyRequireBDFailsFast`: passes; autonomy fails fast when bd is required but missing.

## Evidence details
- Tests are implemented in:
  - `internal/autonomy/smoke_test.go`
  - `internal/autonomy/strictness_test.go`
- Strictness behavior enforced in:
  - `internal/autonomy/controller.go`
- Contract validation for ACTIONS/tasks resolves local schema refs in:
  - `internal/autonomy/tasks.go`

## Notes
- These tests use a fake codex executor and bd stub to keep the smoke run deterministic and offline.
- For a full live run with real `codex`, use `autocodex "<task>"` with `autocodex.yaml` configured, after ensuring required ACTIONS output is emitted.
