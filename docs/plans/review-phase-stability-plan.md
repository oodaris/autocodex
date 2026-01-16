# Review Phase Stability Plan

## Metadata
```yaml
id: review-phase-stability
spec: docs/specs/review-phase-stability.md
owner: autocodex
status: draft
created: 2026-01-16
updated: 2026-01-16
```

## Phases
- Contracts: Update config expectations for per-phase timeouts and prompt guardrail (docs only).
- Code: Stream Codex output, log prompt/snapshot sizes, add per-phase idle overrides and guardrails.
- Tests: Add unit tests for streaming output and review guardrail behavior.
- Docs: Update config examples and troubleshooting notes.
- Rollout: Run an autonomy loop smoke test and capture artifacts.

## Tasks (machine-readable)
- `docs/plans/review-phase-stability-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| autocodex-rps1 | Draft review phase stability plan + tasks | — | todo | Create plan and tasks JSON per template. |
| autocodex-rps2 | Stream codex stdout/stderr to artifacts | autocodex-rps1 | todo | Ensure mid-flight output is persisted. |
| autocodex-rps3 | Add prompt size + snapshot size diagnostics | autocodex-rps2 | todo | Record sizes before review exec. |
| autocodex-rps4 | Add per-phase idle timeouts + review guardrail | autocodex-rps3 | todo | Review can be longer; skip if prompt too large. |
| autocodex-rps5 | Add tests for streaming + guardrails | autocodex-rps4 | todo | Unit tests in Go. |
| autocodex-rps6 | Run loop smoke test and capture evidence | autocodex-rps5 | todo | Confirm review artifacts present. |

## Risks
- Guardrails may prematurely skip valid reviews.
- Longer idle timeouts can slow failure detection.
- Extra artifacts could increase storage footprint.

## Evidence checklist
- `review-stdout.txt`/`review-stderr.txt` or `review-skipped.txt` exists for a review run.
- Prompt size and snapshot size logged in artifacts.
- Tests pass: `go test ./...`.
- Loop smoke run completes without stale finalization.

## Rollout / rollback
- Rollout: enable defaults in `autocodex.yaml`, run smoke test `autocodex "loop smoke test review"`.
- Rollback: revert streaming/guardrail changes and remove new config keys from example config.
