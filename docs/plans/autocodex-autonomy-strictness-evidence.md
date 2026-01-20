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

## Live smoke ACTIONS (2026-01-20)
```json
{
  "version": "1.0",
  "summary": "Live smoke autonomy evidence for autocodex-e36.",
  "next": {
    "type": "none",
    "reason": "Single-bead smoke complete."
  },
  "updates": {
    "beads": [
      {
        "id": "autocodex-e36",
        "status": "done",
        "note": "Spec/plan/tasks artifacts and evidence captured."
      }
    ]
  },
  "gates": {
    "blocking": false,
    "tests": [
      "ACTIONS JSON emitted in evidence doc."
    ],
    "evidence": [
      "docs/plans/autocodex-autonomy-strictness-evidence.md"
    ],
    "verification": [
      "bd show autocodex-e36 reports status done."
    ]
  }
}
```

## Live smoke run (reasoning_effort=low) — 2026-01-20
Config:
- Temporary config used: `/tmp/autocodex-low.yaml` (copied from `autocodex.yaml` with `codex.reasoning_effort: low`).

Command:
```bash
autocodex run --config /tmp/autocodex-low.yaml --task "Live smoke: verify strict autonomy. Generate spec/plan/tasks for a single bead. In the test phase, emit ACTIONS JSON with next.type=none, updates current bead to done, and gates.blocking=false. Capture evidence in docs/plans/autocodex-autonomy-strictness-evidence.md."
```

Results:
- Run ID: `20260120T011444Z-907ebbe2` (completed; 5 phases).
- ACTIONS JSON emitted at `.autocodex/runs/20260120T011444Z-907ebbe2/artifacts/test-final.txt` with `next.type=none` and `gates.blocking=false`.
- Verified `bd show autocodex-e36` reports status `CLOSED`.
- Live smoke doc artifacts were generated for the run and then removed after verification to keep the repo clean.

Notes:
- Autonomy attempted to continue to another ready bead and started run `20260120T012535Z-6e2cb715`; it was stopped to keep the smoke scoped.
