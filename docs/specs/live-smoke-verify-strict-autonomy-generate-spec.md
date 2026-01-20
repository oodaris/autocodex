# Live Smoke: Verify Strict Autonomy

## Metadata
```yaml
id: live-smoke-verify-strict-autonomy-generate-spec
owner: fatih
status: draft
created: 2026-01-20
updated: 2026-01-20
```

## Problem statement
Strict autonomy needs a live smoke validation that proves a single-bead run can generate compliant spec/plan/tasks artifacts and emit a valid ACTIONS payload in the test phase so the loop can close the bead deterministically.

## Goals
- Generate a single-bead spec, plan, and tasks JSON for the live smoke.
- Emit ACTIONS JSON in the test phase with `next.type=none`, the current bead marked `done`, and `gates.blocking=false`.
- Capture evidence in `docs/plans/autocodex-autonomy-strictness-evidence.md`.

## Non-goals
- Changing runtime autonomy logic or schemas.
- Running external services or non-deterministic live executions.
- Producing multi-bead plans.

## Requirements
### Functional
- Create `docs/specs/live-smoke-verify-strict-autonomy-generate-spec.md` using the spec template.
- Create a plan doc and a tasks JSON file for a single bead that conform to repo templates/schemas.
- Update the autonomy strictness evidence doc with the required ACTIONS JSON.
- Close the bead in bd once evidence is captured.

### Non-functional
- Doc-only changes; no code or contract updates.
- Outputs remain ASCII and repo-local.
- Autonomy mode assumptions captured explicitly in the spec.

## Interfaces / data contracts
- Tasks JSON must conform to `docs/contracts/autonomy-tasks.schema.json`.
- ACTIONS JSON must conform to `docs/contracts/autonomy-actions.schema.json`.
- Bead status updates must use `bd`.

## Acceptance criteria
- Spec, plan, and tasks files exist and reference the same slug.
- Evidence doc includes a compliant ACTIONS JSON block with required fields.
- Bead `autocodex-e36` is marked `done` in bd.

## Open questions
- Assumptions (autonomy mode): Use bead `autocodex-e36` as the single bead; no code/tests are required beyond emitting ACTIONS JSON in evidence; appending to the existing evidence doc is acceptable.

## Risks
- A doc-only smoke may not catch runtime regressions.
- If a different bead was expected, the evidence may need rework.

## References
- `docs/specs/TEMPLATE.md`
- `docs/plans/TEMPLATE.md`
- `docs/contracts/autonomy-tasks.schema.json`
- `docs/contracts/autonomy-actions.schema.json`
- `docs/plans/autocodex-autonomy-strictness-plan.md`
