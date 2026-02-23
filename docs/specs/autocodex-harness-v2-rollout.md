# Autocodex Harness v2 Rollout

## Problem
`autocodex` already supports autonomy, bead-parallel execution, and collaboration presets, but it lacks a deterministic harness policy pack with executable preflight/lint/eval gates. This leaves a gap between "run completed" and "release-ready with evidence".

## Goal
Adopt a Harness-v2-style workflow in `autocodex` with:
1. a repo-local `.codex` role pack,
2. executable governance (`autocodex harness preflight`, harness config lint, deterministic eval docs),
3. runtime enforcement for high-impact gates,
4. lifecycle/admission metadata for traceability.

## Success Criteria
1. Harness policy pack exists and is validated by lint.
2. `autocodex harness preflight --strict` passes in a healthy environment.
3. High-impact mode blocks closure without council GREEN + critic GO + quality gate pass.
4. Run events include lifecycle/admission metadata fields.
5. RFC backlog is represented as BD tasks with dependencies and evidence requirements.

## In Scope
- CLI/config/runtime/docs/test changes required for the three-phase rollout.
- BD-only strict tracking mode and evidence (no Jira/Tempo dependency).

## Out of Scope
- Mandatory Jira/Tempo closure enforcement.
- External deployment/platform changes.

## Constraints
- Keep backward compatibility when harness mode is disabled.
- Keep coordinator-based parallelism as the deterministic outer scheduler.
- Treat Codex `multi_agent` as optional acceleration, not a hard correctness dependency.
