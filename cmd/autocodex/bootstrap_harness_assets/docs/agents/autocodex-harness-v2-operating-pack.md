# Autocodex Harness v2 Operating Pack

Status: baseline for `autocodex` (2026-02-23)
Owner: platform core
Tracking mode: `BD_STRICT`

## 0) Max capability defaults (project scoped)

This pack favors high capability while keeping deterministic closure gates:
1. Role pack declared in `.codex/config.toml`.
2. Multi-agent enabled for orchestrator/architect roles; specialist roles remain single-agent.
3. Harness mode is policy-driven by `autonomy.harness` runtime config.
4. Governance is executable: preflight + lint + eval docs.
5. The repo-local `max_capability` profile and Harness V2 role files map to `gpt-5.4`; Spark stays isolated in its own profile.

## 1) Canonical decision precedence
1. Nearest `AGENTS.md` wins.
2. Golden workflow is mandatory: Plan -> Contracts -> Code -> Tests -> Docs -> Rollout.
3. BD is execution source of truth.
4. In this repo, release closure uses BD evidence (`BD_STRICT`), not Jira/Tempo.

## 2) Deterministic skill matrix
| Phase | Mandatory skill(s) | Hard gate |
| --- | --- | --- |
| Intake synthesis | `$core-qna-synthesis` | Goals + success criteria explicit |
| Planning/tracking | `$core-holistic-planning-and-tracking` | BD task graph + file scope present |
| Implementation | `$core-test-driven-development` | Failing -> passing evidence for changed behavior |
| Quality gate | `$eng-smart-test-runner` | Non-bypassable gate matrix completed |
| Closure | `$eng-conventional-commit-helper` | Single-intent commits + evidence checklist complete |

## 3) Orchestration pattern A (full workflow)
1. requirements clarifier
2. design strategist
3. tracking operator (create/claim BD tasks and file scopes)
4. parallel council (high-impact only)
5. executors (backend/frontend as needed)
6. quality gate runner
7. independent critic
8. commit curator
9. release evidence operator

Rules:
- No coding before ambiguities are resolved.
- No completion without gate evidence.
- High-impact mode requires council GREEN and critic GO before closure.

## 4) Orchestration pattern E (council gate)
1. Lock assumptions/acceptance criteria.
2. Run independent parallel council members.
3. Anonymize and score using fixed rubric.
4. If verdict not GREEN, remediate and re-run one round.
5. Block closure until GREEN.

## 5) Non-bypassable gate stack
1. `go test ./...`
2. `go vet ./...`
3. `autocodex harness preflight --strict`
4. `autocodex harness lint`

Source-checkout note: when `autocodex.yaml` is absent, the harness commands use
repo-root `config.example.yaml` unless `--config` or `AUTOCODEX_CONFIG` is set.

If any gate fails: stop, produce minimal repro, and create/claim fix bead.

## 6) Runtime lifecycle/admission contract
Required metadata on orchestration transitions:
- `run_id`
- `idempotency_key`
- `attempt`
- `admission_decision`
- `queue_state`
- `retry_backoff_ms`
- `thread_id`, `turn_id`, `item_id`

Required lifecycle events:
- `thread_started`, `turn_started`, `item_started`, `item_completed`, `turn_completed`, `thread_completed`, `thread_failed`

## 7) High-impact trigger criteria
Treat changes as high-impact when they affect:
1. runtime autonomy contracts/schemas,
2. gate or stop-condition semantics,
3. coordinator behavior or bead state transitions,
4. admission/lifecycle metadata guarantees.

## 8) Adoption checklist
1. `.codex/config.toml` and required role files exist.
2. `autocodex harness lint` passes.
3. `bash scripts/dev/harness-cli-preflight.sh` passes.
4. RFC backlog exists in `docs/plans/*-tasks.json` and BD tasks with dependencies.
5. High-impact changes include council + critic gate evidence.
