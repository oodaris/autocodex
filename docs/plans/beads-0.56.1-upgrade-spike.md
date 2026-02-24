# Beads 0.56.1 Upgrade Spike Plan

## Metadata
```yaml
id: beads-0.56.1-upgrade-spike
spec: docs/specs/beads-0.56.1-upgrade-spike.md
owner: platform-core
status: done
created: 2026-02-24
updated: 2026-02-24
```

## Summary
This plan executes a focused Beads `v0.56.1` compatibility spike for `autocodex`. The spike is intentionally constrained to docs and readiness checks (`doctor` + `harness preflight`) and does not change autonomy runtime BD command paths.

## Phases

### Phase 0: Tracking + Planning Artifacts
- Create/update spec + plan + tasks JSON artifacts for the spike.
- Create BD epic + child tasks with dependency chain and file scopes.
- Align acceptance criteria and evidence checklist with docs + readiness scope.

### Phase 1: Doctor + Harness Preflight Deltas
- Add Beads version check (target `>=0.56.1`) in doctor.
- Add Beads Dolt readiness check via `bd dolt show` in doctor.
- Ensure harness preflight strict mode escalates Beads readiness warnings.
- Add/extend tests for version parsing and Dolt readiness parsing.

### Phase 2: Documentation + Runbook Consolidation
- Remove stale `bd sync commits/pushes` guidance.
- Add Beads `0.56.1` operator guidance and validation commands.
- Add a dedicated Beads `0.56.1` runbook and align existing runbooks.

## Tasks (machine-readable)
- `docs/plans/beads-0.56.1-upgrade-spike-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.

## Must-have gates (autonomy)
- Tests required:
  - `go test ./cmd/autocodex`
  - `go test ./...`
  - `go vet ./...`
- Runtime verification required:
  - `go run ./cmd/autocodex doctor --config config.example.yaml --strict`
  - `go run ./cmd/autocodex harness preflight --config config.example.yaml --strict`
  - `python3 scripts/harness_config_lint.py`
- Evidence required (paths):
  - `docs/plans/beads-0.56.1-upgrade-spike.md`
  - `docs/specs/beads-0.56.1-upgrade-spike.md`
  - `docs/runbooks/beads-0.56.1-upgrade.md`

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| beads-0561-0 | Phase 0 tracking + plan artifacts | — | done | Spec/plan/tasks JSON + BD dependency graph created. |
| beads-0561-1 | Doctor Beads checks | beads-0561-0 | done | Added `bd-version` + `bd-dolt` checks with parser tests. |
| beads-0561-2 | Harness preflight deltas | beads-0561-1 | done | Strict-mode blocking warnings + shell preflight version gate. |
| beads-0561-3 | Docs + runbook updates | beads-0561-2 | done | Removed stale sync claims; added Beads 0.56.1 runbook + server note. |
| beads-0561-4 | Final verification + closure evidence | beads-0561-3 | done | Gates passed; BD chain closed. |

## Risks
- Local environments on Beads `<0.56.1` may fail strict preflight after this spike.
- `bd dolt show` output format changes could reduce parser reliability.
- Stale operational habits around `bd sync` may persist without explicit runbook updates.

## Evidence checklist
- [x] Spec/plan/tasks artifacts created and linked.
- [x] Doctor includes Beads version + Dolt readiness checks.
- [x] Harness preflight strict mode enforces Beads readiness findings.
- [x] Docs and runbooks updated for Beads `0.56.1` guidance.
- [x] Verification gates pass and are recorded.
- [x] BD tasks closed and `bd sync` executed.

## Verification evidence
- `go test ./...` -> pass (2026-02-24).
- `go vet ./...` -> pass (2026-02-24).
- `python3 scripts/harness_config_lint.py` -> pass (2026-02-24).
- `go run ./cmd/autocodex doctor --config config.example.yaml --strict` -> pass after creating `.autocodex/memory`.
- `go run ./cmd/autocodex harness preflight --config config.example.yaml --strict` -> pass.
- `bash scripts/dev/harness-cli-preflight.sh` -> pass with `bd 0.56.1`.

## Rollout / rollback
- Rollout:
  1. Merge spike branch after verification.
  2. Communicate Beads `0.56.1` requirement in release/process docs.
  3. Use strict preflight for high-impact autonomy sessions.
- Rollback:
  1. Revert doctor/preflight Beads checks if they cause unacceptable friction.
  2. Revert runbook/docs updates in one commit if guidance must return to previous baseline.
  3. Open follow-up bead for full runtime compatibility layer if needed.
