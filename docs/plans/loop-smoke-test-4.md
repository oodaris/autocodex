# Loop Smoke Test 4 (Harness + Schema)

## Purpose
Provide a deterministic, offline smoke test for the autonomy loop that:
- exercises the end-to-end harness (fake bd + templates + config)
- validates generated tasks JSON against the autonomy schema

This is the baseline autonomy smoke check used before releases and rollout changes.

## What it covers
- `internal/autonomy/smoke_harness_test.go` (shared harness for autonomy tests)
- `internal/autonomy/smoke_test.go` (basic autonomy loop smoke run)
- `internal/autonomy/smoke_integration_test.go` (schema validation for tasks JSON)

## How to run
```bash
go test ./internal/autonomy
```

To focus only on the smoke harness + schema test:
```bash
go test ./internal/autonomy -run TestAutonomyLoopSmokeTasksSchema
```

## Expected outcomes
- Smoke tests pass on macOS/Linux.
- Tasks JSON validates against `docs/contracts/autonomy-tasks.schema.json`.
- Bead status recorded as `done` by the fake `bd` stub.

## Rollout hook
Before shipping autonomy changes or releases:
1) Run the smoke tests above.
2) Capture evidence in the release checklist or rollout notes.
