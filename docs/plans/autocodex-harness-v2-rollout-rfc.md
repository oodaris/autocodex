# Autocodex Harness v2 Rollout RFC

## Metadata
```yaml
id: autocodex-harness-v2-rollout
spec: docs/specs/autocodex-harness-v2-rollout.md
owner: platform-core
status: implemented
created: 2026-02-23
updated: 2026-02-23
```

## Summary
This RFC defines a phased rollout to apply Harness v2 flow to `autocodex` end-to-end. It introduces a repo-local `.codex` role pack, executable harness governance (preflight + lint + eval assets), and runtime/contract enforcement for high-impact tasks.

Tracking mode for this rollout is `BD_STRICT`: closure is blocked unless BD state + evidence checklist are complete.

## Phases

### Phase 0: Governance and Policy Pack Foundation
- Add rollout spec + this RFC + machine-readable task list.
- Add `.codex/config.toml` and role configs for orchestrator, council, critic, gates, and release flow.
- Add operating pack runbook, deterministic eval docs, and executable harness lint/preflight checks.
- Initialize and populate BD task graph with file scopes + dependencies.

### Phase 1: Product Integration (CLI/config/doctor)
- Add `autocodex harness preflight` command.
- Add `autonomy.harness` config section, defaults, and validation.
- Extend config schema/docs/examples.
- Upgrade `autocodex doctor` with feature-aware Codex capability checks.

### Phase 2: Runtime Enforcement and Traceability
- Extend autonomy action schema and runtime gates for high-impact council/critic/quality requirements.
- Add lifecycle/admission metadata to run events and runtime traces.
- Add/extend tests covering strictness and backward compatibility.

## Concrete Diff Plan (Executed)

### Phase 0 backlog + acceptance criteria
- Backlog:
  - RFC/spec/tasks artifacts (`docs/specs/*`, `docs/plans/*`).
  - Harness role pack and policy config (`.codex/config.toml`, `.codex/agents/*.toml`).
  - Harness governance assets (`autocodex harness lint`, `scripts/dev/harness-cli-preflight.sh`, runbooks/eval docs).
  - BD rollout epic + dependency graph.
- Acceptance criteria:
  - [x] Harness role pack exists and passes `autocodex harness lint`.
  - [x] Preflight script and runbook exist with deterministic success marker.
  - [x] BD graph created with dependencies and lintable issue descriptions.

### Phase 1 backlog + acceptance criteria
- Backlog:
  - Add `autocodex harness preflight`.
  - Add `autonomy.harness` config model/defaults/validation.
  - Extend schema/examples/docs.
  - Add codex feature-capability checks in doctor.
- Acceptance criteria:
  - [x] `autocodex harness preflight --strict` path exists and executes doctor + lint checks.
  - [x] Invalid `autonomy.harness.*` values fail validation.
  - [x] Config schema/examples/docs include `autonomy.harness`.
  - [x] Doctor reports required/recommended Codex feature availability.

### Phase 2 backlog + acceptance criteria
- Backlog:
  - Extend ACTIONS schema/gates for high-impact closure semantics.
  - Enforce council/critic/quality gate policy in runtime.
  - Emit lifecycle/admission metadata with thread/turn/item events.
  - Add coverage for success and failure lifecycle traces.
- Acceptance criteria:
  - [x] High-impact closure fails without `council_verdict=GREEN`, `critic_verdict=GO`, `quality_gate_passed=true`.
  - [x] Successful runs emit `thread_completed` and `run_complete` with lifecycle metadata contract fields.
  - [x] Failed runs emit `thread_failed` lifecycle traces with metadata contract fields.
  - [x] Backward-compatible behavior remains for non-harness/normal impact runs.

## Tasks (machine-readable)
- `docs/plans/autocodex-harness-v2-rollout-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.

## Must-have gates (autonomy)
- Tests required:
  - `go test ./...`
  - `go vet ./...`
- Runtime verification required:
  - `autocodex harness preflight --strict`
  - `autocodex harness lint`
- Evidence required (paths):
  - `docs/plans/autocodex-harness-v2-rollout-rfc.md`
  - `docs/specs/autocodex-harness-v2-rollout.md`
  - `docs/agents/autocodex-harness-v2-operating-pack.md`

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| autocodex-hv2-0 | Draft RFC + spec + tasks | — | done | Scope, phases, gates, and evidence defined. |
| autocodex-hv2-1 | Add .codex harness role pack | autocodex-hv2-0 | done | Role configs + operating pack implemented. |
| autocodex-hv2-2 | Add preflight + harness lint + eval docs | autocodex-hv2-1 | done | Governance checks and eval docs implemented. |
| autocodex-hv2-3 | Add config/model/schema docs for harness | autocodex-hv2-0 | done | `autonomy.harness` config surface shipped. |
| autocodex-hv2-4 | Add `autocodex harness preflight` + doctor feature checks | autocodex-hv2-3,autocodex-hv2-2 | done | Capability-aware readiness checks shipped. |
| autocodex-hv2-5 | Enforce high-impact council/critic/quality gates | autocodex-hv2-4 | done | Runtime gate enforcement shipped with tests. |
| autocodex-hv2-6 | Add lifecycle/admission event metadata | autocodex-hv2-5 | done | Thread/turn/item lifecycle events shipped. |
| autocodex-hv2-7 | Final verification and rollout docs | autocodex-hv2-6 | done | Gate matrix verified and docs updated. |

## Risks
- Harness mode could add execution overhead.
- Strict gates may initially block many runs until prompts are tuned.
- Codex feature drift can invalidate assumptions if checked only by semver.

## Evidence checklist
- [x] All Phase 0/1/2 files and tests updated.
- [x] `autocodex harness preflight --strict` passing evidence captured.
- [x] `go test ./...` and `go vet ./...` pass.
- [x] BD tasks updated with statuses and dependency closure.

## Verification Evidence (2026-02-23)
- `go test ./...` -> pass.
- `go vet ./...` -> pass.
- `autocodex harness lint` -> pass.
- `bash scripts/dev/harness-cli-preflight.sh` -> pass.
- `go run ./cmd/autocodex harness preflight --config config.example.yaml --strict` -> pass.

## Rollout / rollback
- Rollout:
  1. Merge with `autonomy.harness.enabled: false` default.
  2. Enable in pilot config for selected runs.
  3. Promote to broader usage after stable gate pass trends.
- Rollback:
  1. Disable harness mode in config.
  2. Keep role pack/docs/scripts in repo as non-enforcing assets.
  3. Revert runtime enforcement deltas only if regressions appear.
