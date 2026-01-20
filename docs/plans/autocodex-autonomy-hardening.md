# Autonomy hardening (runs, validation, recovery)

## Metadata
```yaml
id: autocodex-autonomy-hardening
spec: docs/specs/autocodex-autonomy-hardening.md
owner: maintainer
status: draft
created: 2026-01-20
updated: 2026-01-20
```

## Phases
- Contracts: config/schema flags, run metadata shapes, API response contracts.
- Code: run registry + persistence, ACTIONS validation, deterministic artifacts, preflight checks.
- Tests: unit + integration for run listing, resume safety, schema enforcement.
- Docs: CLI + troubleshooting updates, autonomy expectations.
- Rollout: gated by config flags; safe defaults for public usage.

## Tasks (machine-readable)
- `docs/plans/autocodex-autonomy-hardening-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.

## Must-have gates (autonomy)
- Tests required:
  - `go test ./...`
  - new unit tests for run registry, resume safety, ACTIONS validation.
- Runtime verification required:
  - `autocodex run` + `autocodex status` shows run state.
  - `autocodex ui` shows run list with summary.
- Evidence required (paths):
  - `docs/evidence/autonomy-hardening/` (logs + screenshots)

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| autocodex-1e3 | Run registry + status persistence | - | todo | Persist run metadata for UI/status/resume. |
| autocodex-0ci | ACTIONS validation + strict gating | autocodex-1e3 | todo | Fail/mark-degraded on invalid ACTIONS. |
| autocodex-4jf | Schema policy flags (fail vs fallback) | - | todo | Config-driven strictness. |
| autocodex-qxr | Preflight checks / doctor command | - | todo | Tooling + repo sanity checks. |
| autocodex-ry3 | Deterministic artifact paths | autocodex-1e3 | todo | Avoid collisions across runs. |
| autocodex-o1f | Resume safety + run selection | autocodex-1e3 | todo | Resume only unfinished runs unless forced. |
| autocodex-vbr | Observability: run_id + stage logs | - | todo | Consistent structured logs. |
| autocodex-yju | UI hardening for runs payload | autocodex-1e3 | todo | Guard against invalid API payloads. |

## Risks
- Run registry introduces schema/backward-compatibility issues for existing runs.
- Strict validation may increase failed runs without clear remediation paths.
- Preflight checks could block valid edge cases unless override exists.

## Evidence checklist
- Run listing and status visible in UI.
- Resume refuses completed runs unless `--force`.
- ACTIONS invalid payloads fail fast with clear error.

## Rollout / rollback
- Default to safe-but-useful: `autonomy.fail_on_schema_error=true`, `autonomy.allow_fallback_tasks=true`.
- Provide `--no-preflight` override for expert usage.
- Rollback by reverting config defaults and disabling new checks.
