# Beads 0.56.1 Upgrade Spike

## Problem
`autocodex` currently operates with local Beads `0.55.4`, but upstream Beads `0.56.1` introduces operational changes that affect repo workflows: stricter server-mode assumptions and changed `bd sync` semantics. Existing docs and readiness checks in this repo still contain stale guidance that can mislead operators.

## Goal
Deliver a focused spike that makes `autocodex` explicitly ready for Beads `v0.56.1` in documentation and readiness tooling:
1. update docs/runbooks for current Beads semantics,
2. add doctor/preflight checks for Beads version and Dolt readiness,
3. preserve runtime behavior for autonomy/bead orchestration (no execution-path refactor in this spike).

## Success Criteria
1. `autocodex doctor` reports `bd-version` and `bd-dolt` readiness results.
2. `autocodex harness preflight --strict` fails fast on Beads version mismatch and Dolt server readiness problems.
3. Repo docs no longer claim `bd sync` commits/pushes to `beads-sync`.
4. A clear runbook exists for Beads `0.56.1` operator setup/verification.
5. Phase 0/1/2 work is tracked in BD with dependencies and acceptance criteria.

## In Scope
- Doctor and harness preflight deltas.
- Documentation and runbook updates for Beads `0.56.1`.
- BD planning/tracking artifacts for this spike.

## Out of Scope
- Refactoring autonomy runtime BD command execution paths.
- Introducing a Beads multi-version runtime compatibility layer.
- Jira integration automation changes.

## Constraints
- Keep changes backward-safe for users still on older Beads; strict mode should enforce, non-strict should inform.
- Do not manually edit `.beads` data files; use `bd` commands.
- Keep the spike narrowly scoped to docs + readiness checks.

