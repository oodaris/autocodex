# Autocodex Engineering Playbook

This repo contains an open-source Codex autorunner with a Go CLI + local API and a React/Vite UI.

## Skills invocation
- Use **core-qna-synthesis** for multi-part or ambiguous questions.
- Use **core-holistic-planning-and-tracking** for planning and Beads creation.
- Use **core-ask-questions-if-underspecified** when required inputs are missing.
- Use **eng-code-review-playbook** for formal reviews.

## 0) Beads-first workflow (bd)
- Use `bd` as the single source of truth for tasks and coordination.
- Do not edit `.beads/` manually; use the `bd` CLI.
- Create tasks with `bd create "<title>"`; add dependencies with `bd dep add <task> <depends_on>`.
- Each task must declare a **Files** scope; do not edit outside of the claim.

### bd task template
```md
Title:
Owner:
Status: todo | in_progress | review | done
Goal:
Scope:
Files:
Dependencies:
Constraints:
Plan:
Acceptance Criteria:
Contracts:
Code:
Tests:
Docs:
Rollout/Rollback:
Observability:
Notes:
```

## 1) Golden workflow
Plan → Contracts → Code → Tests → Docs → Rollout

## 2) Project overview
- **CLI**: Go (Cobra) for orchestration and Codex CLI integration.
- **Local API**: Go HTTP server for status/events/UI integration.
- **UI**: React + Vite (deployable to Vercel).
- **Plugins**: External processes with a manifest + RPC protocol (cross-platform, versioned).
- **State**: Repo-local markdown memory + JSONL logs.

## 3) Commands (initial)
- Go tests: `go test ./...`
- Go lint (when configured): `golangci-lint run ./...`
- UI dev (when configured): `cd web && npm i && npm run dev`
- UI build: `cd web && npm run build`

## 4) Observability
- Structured JSON logs with: `trace_id, route, status, latency_ms` (+ task/bead ids when applicable).
- Emit RED/USE metrics where meaningful (API + worker loop).

## 5) Security
- No secrets in repo; use `.env` and document vars in `.env.example`.
- Yolo mode must be explicit in config and surfaced in UX/CLI warnings.
