# live smoke verify strict autonomy plan

## Metadata
```yaml
id: live-smoke-verify-strict-autonomy-generate-spec
spec: docs/specs/live-smoke-verify-strict-autonomy-generate-spec.md
owner: fatih
status: draft
created: 2026-01-20
updated: 2026-01-20
```

## Phases
- Plan: generate spec/plan/tasks artifacts for a single bead.
- Contracts: validate tasks/actions references against existing schemas.
- Code: N/A (doc-only).
- Tests: emit required ACTIONS JSON in the evidence log.
- Docs: update spec/plan/tasks and evidence files.
- Rollout: close bead and sync status.

## Tasks (machine-readable)
- `docs/plans/live-smoke-verify-strict-autonomy-generate-spec-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.

## Must-have gates (autonomy)
- Tests required:
  - ACTIONS JSON emitted in test phase (documented in evidence).
- Runtime verification required:
  - `bd show autocodex-e36` shows status `done`.
- Evidence required (paths):
  - `docs/plans/autocodex-autonomy-strictness-evidence.md`

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| autocodex-e36 | Live smoke: verify strict autonomy (spec/plan/tasks + evidence) | - | in_progress | Single-bead autonomy smoke artifacts. |

## Risks
- Doc-only smoke may miss runtime regressions.

## Evidence checklist
- `docs/specs/live-smoke-verify-strict-autonomy-generate-spec.md`
- `docs/plans/live-smoke-verify-strict-autonomy-generate-spec-plan.md`
- `docs/plans/live-smoke-verify-strict-autonomy-generate-spec-tasks.json`
- `docs/plans/autocodex-autonomy-strictness-evidence.md`

## Rollout / rollback
- Rollout: close bead `autocodex-e36`, run `bd sync` at session end.
- Rollback: reopen bead and revert doc updates if needed.
