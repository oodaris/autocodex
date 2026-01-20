# autocodex autonomy loop plan

## Problem statement
Extend autocodex to run a fully autonomous, bead‑driven loop: given a single task, it should generate spec + plan + beads, execute beads in dependency order, run review + tests, and continue until all beads are complete. The loop must be deterministic, auditable, and safe under yolo mode defaults.

## Success criteria
- `autocodex "task"` can generate:
  - `docs/specs/<slug>.md`
  - `docs/plans/<slug>-plan.md`
  - beads in `.beads/` with dependencies
- Bead execution is dependency‑aware and terminates when no open beads remain.
- Each bead run includes review and tests with gating.
- Codex output is parsed into actionable “next bead” instructions.
- Failures create fix‑beads and retry until green or stop conditions trigger.

## Scope
### In scope
- Spec + plan generation using Codex skills.
- Plan → tasks translation and bead creation.
- Bead scheduler + lifecycle automation (open → in_progress → done).
- Action parsing from Codex outputs to drive next steps.
- Test and review gating (auto‑fix beads on failures).
- CLI integration for `autocodex "task"` shortcut.
- Docs updates for autonomy mode.

### Out of scope (this phase)
- Multi‑repo orchestration or parallel bead execution.
- External workflow engines (Inngest/Celery).
- Cloud deployment.

## Constraints
- Must use `bd` CLI (no manual edits to `.beads/`).
- Keep the loop deterministic and observable.
- Default to local execution (Codex CLI).

## Architecture overview
- **Autonomy controller**: orchestrates spec → plan → beads → bead execution.
- **Action parser**: extracts structured `ACTIONS` block from Codex output.
- **Bead scheduler**: chooses next bead by dependency order.
- **Gates**: review + test run after each bead.
- **Artifacts**: spec/plan documents + task JSON.

## Phases & dependencies

### Phase A — Contracts + templates
- Define spec + plan templates and a structured tasks schema.
- Define `ACTIONS` schema for Codex output.

### Phase B — Config + CLI integration
- Add autonomy config block.
- Hook `autocodex "task"` into autonomy mode when enabled.

### Phase C — Spec/plan generation
- Run skills to generate spec + plan artifacts.
- Persist task JSON for bead creation.

### Phase D — Bead creation + scheduler
- Create beads with dependencies.
- Pick next bead based on `bd ready`.

### Phase E — Action parsing + gating
- Parse Codex output into next actions.
- Run review + test gates and auto‑create fix beads.

### Phase F — Docs + smoke tests
- Document autonomy mode in README + AGENTS.
- Add smoke test for autonomy run (dry‑run mode).
- Loop smoke test 4 harness + schema validation: `docs/plans/loop-smoke-test-4.md`.

## Risks & mitigations
- **LLM output drift** → enforce strict `ACTIONS` schema; validate before applying.
- **Dependency deadlocks** → fail fast when no ready beads; surface graph.
- **Test flakiness** → allow retry budget + explicit stop conditions.

## Evidence checklist
- Spec + plan artifacts generated from a single task.
- Beads created with dependencies and proper lifecycle updates.
- Logs show bead selection, actions parsed, and gate outcomes.
- Tests pass in CI for autonomy flow.

## Rollout/rollback
- Feature‑flag autonomy mode (config).
- Roll back by disabling autonomy block and using legacy run loop.
- Rollout hook: run the loop smoke tests in `docs/plans/loop-smoke-test-4.md` before release.
