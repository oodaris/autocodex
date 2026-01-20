# Autocodex OSS DX + Refactor Plan

## Metadata
```yaml
id: autocodex-oss-dx-refactor
owner: fatih
status: draft
created: 2026-01-20
updated: 2026-01-20
```

## Problem statement
We want a stronger OSS developer experience for autocodex (docs, linting, badges, troubleshooting) and to reduce maintenance risk by splitting several oversized Go files.

## Goals
- Provide clear CLI/docs/troubleshooting for OSS users.
- Add CI checks for lint/static analysis and vulnerability scanning.
- Add standard OSS badges once CI coverage/scorecard are in place.
- Refactor long Go files into smaller, domain-focused files without changing behavior.

## Non-goals
- No new features in autonomy logic.
- No API contract changes.
- No UI redesign work.

## Phases
1) **Docs + OSS signals** (CLI doc, troubleshooting, badges)
2) **Quality gates** (golangci-lint or staticcheck + govulncheck)
3) **Refactors** (state/orchestrator/bootstrap/config/api)
4) **Validation** (go test ./..., lint, and any updated docs)

## Task list (human summary)
| id | title | deps | status | notes |
| --- | --- | --- | --- | --- |
| autocodex-0d8 | Doc comments for public surfaces |  | todo | Only for exported API surfaces intended for users |
| autocodex-5fg | Add docs/CLI.md |  | todo | Single source for commands/flags |
| autocodex-190 | Add troubleshooting section |  | todo | address in use/auth/bd missing |
| autocodex-r1l | Add lint + govulncheck in CI |  | todo | Decide golangci-lint vs staticcheck |
| autocodex-tcv | Add README badges | autocodex-r1l | todo | CI, coverage, Go Report Card, Scorecard |
| autocodex-adb | Split internal/state/state.go |  | todo | Keep behavior identical |
| autocodex-0ur | Split internal/orchestrator/orchestrator.go |  | todo | Keep behavior identical |
| autocodex-2ub | Split cmd/autocodex/bootstrap.go |  | todo | Keep behavior identical |
| autocodex-aut | Split internal/config/config.go |  | todo | Keep behavior identical |
| autocodex-5bn | Split internal/api/api.go |  | todo | Keep behavior identical |

## Evidence checklist
- `go test ./...`
- CI lint + govulncheck passing
- README updated with badges + docs links
- docs/CLI.md and troubleshooting section published

## Rollout / rollback
- Rollout: merge in phases (docs → CI → refactors). Release after refactors stabilize.
- Rollback: revert the specific refactor or CI change; docs changes are safe to keep.
