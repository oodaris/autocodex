# autocodex autonomy strictness plan

status: draft
owner: maintainer
created: 2026-01-20

## Problem statement
Autonomy runs can complete without emitting ACTIONS or satisfying runtime validation gates, which leads to premature bead completion and weak continuity across runs. We need strict, autonomy-only guardrails so the loop fails closed when outputs are non-compliant, while keeping manual/non-autonomy usage flexible.

## Goals
- Require ACTIONS output for autonomy runs and enforce gates deterministically.
- Default feedback context on in autonomy mode to improve continuity.
- Make bead selection deterministic and explicit for multi-bead plans.
- Define must-have gates (tests, runtime verification, evidence) in templates and contracts.

## Non-goals
- Changing the core phase ordering or adding new phases.
- Altering non-autonomy behavior or CLI UX beyond documentation.

## Scope
- Config + contracts updates for strict autonomy.
- Autonomy controller updates for ACTIONS enforcement and gate handling.
- Documentation updates for plan templates and guardrails.
- Tests + a smoke validation run with evidence.

## Plan (phases)
### 1) Contracts + config
- Add autonomy strictness keys and defaults.
- Update autonomy schemas to encode gates/evidence requirements.

### 2) Code
- Enforce ACTIONS-required behavior in autonomy controller.
- Enforce bd availability when required.
- Require explicit Next when multiple beads are ready (strict mode).

### 3) Tests
- Unit tests for ACTIONS-required and feedback defaults.
- Validate strictness failure modes (missing actions, missing bd).

### 4) Docs
- Update plan template with must-have gate checklist.
- Update README/AGENTS/config docs with strictness behavior.

### 5) Rollout
- Run an autonomy smoke loop and capture evidence.
- Record results and any rollback steps.

## Work breakdown (BD)
| Task ID | Title | Depends on | Outcome |
| --- | --- | --- | --- |
| autocodex-stt | Autonomy strictness: contracts + config defaults | - | Config + contract updates defined and documented. |
| autocodex-4y7 | Autonomy controller: require ACTIONS + gate handling | autocodex-stt | Runtime enforcement of strict autonomy behavior. |
| autocodex-p9q | Autonomy gating + plan template updates | autocodex-stt | Templates/docs encode must-have gates. |
| autocodex-dh2 | Autonomy tests: actions required + feedback defaults | autocodex-stt, autocodex-4y7, autocodex-p9q | Regression coverage for strictness. |
| autocodex-wi8 | Autonomy rollout: smoke validation + evidence | autocodex-dh2 | Evidence captured for strict loop behavior. |

## Acceptance criteria
- Autonomy runs fail closed if ACTIONS are missing or invalid.
- feedback context defaults to on for autonomy runs and respects caps.
- Plans/templates include explicit gate requirements and evidence paths.
- Tests cover strictness behavior and defaults.
- Evidence captured for a smoke run.

## Evidence checklist
- Config + contract diffs for strictness settings.
- Unit test results for strictness behavior.
- Smoke run logs + artifacts.
- Evidence doc: `docs/plans/autocodex-autonomy-strictness-evidence.md`.

## Risks and mitigations
- **Risk**: Strictness blocks autonomy on older skills that don’t emit ACTIONS.
  - **Mitigation**: Provide clear error messaging and a temporary override flag.
- **Risk**: Feedback context increases prompt size.
  - **Mitigation**: Keep max_bytes caps; allow opt-out in config.

## Rollout / rollback
- Rollout: enable strictness in `autocodex.yaml` (autonomy-only), run smoke task.
- Rollback: disable strictness flags and revert contract changes if needed.
