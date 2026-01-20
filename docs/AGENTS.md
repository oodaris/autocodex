# autocodex Engineering Playbook

This repo contains autocodex: a Codex runner with a Go CLI, local API, and a React/Vite UI.

AGENTS.md is the agent-focused companion to README. Keep it short, prescriptive, and aligned with the golden workflow.

## Skills invocation
- Use **core-qna-synthesis** for multi-part or ambiguous questions.
- Use **core-holistic-planning-and-tracking** for planning and Beads creation.
- Use **core-ask-questions-if-underspecified** when required inputs are missing.
- Use **eng-code-review-playbook** for formal reviews.
- Prefer repo-local skills in `skills/` over external/private skills.

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
- **CLI**: Go (standard library flags) for orchestration and Codex CLI integration.
- **Local API**: Go HTTP server for status/events/UI integration.
- **UI**: React + Vite (deployable to Vercel).
- **Plugins**: External processes with a manifest + JSON-RPC (stdio) protocol.
- **State**: Repo-local markdown memory + JSONL logs.

## 2.1) Agent setup (minimum)
- For autonomy-ready setup, run `autocodex bootstrap` (creates config, templates/schemas, and a minimal skill pack).
- For minimal setup, run `autocodex init` (config + `.autocodex/` only).
- If `config.example.yaml` is missing, bootstrap falls back to the embedded config.
- Codex CLI available on PATH or set `codex.cli_path` in config.
- Skill paths configured under `skills.paths` (bootstrap writes skills into `skills/`).
- Optional: `bd` installed for bead tracking (missing `bd` is a warning; autonomy still runs).

Common start:
```bash
autocodex "Add a quick summary to memory docs."
```

## 2.2) Autonomy loop notes
- Autonomy mode generates spec/plan/tasks and creates beads from the plan.
- The **test** phase should emit an `ACTIONS` JSON block (per `docs/contracts/autonomy-actions.schema.json`) so autocodex can update bead status and select the next bead.
- Gate failures stop the loop and auto-create a fix bead when enabled.
- Plans must include explicit must-have gates (tests, runtime verification, evidence paths) so autonomy can enforce completion.

## 3) Commands
- Go tests: `go test ./...`
- Go vet: `go vet ./...`
- Go fmt: `gofmt -w $(rg --files -g '*.go')`
- UI dev (when configured): `cd web && npm i && npm run dev`
- UI build: `cd web && npm run build`

## 4) Observability
- Structured JSON logs with: `trace_id, tenant_id, route, status, latency_ms`.
- Log to stderr for diagnostics; stdout reserved for primary command output.

## 5) Safety
- No secrets in repo; use `.env` and document vars in `.env.example`.
- Yolo mode must be explicit in config and surfaced in UX/CLI warnings.
- Local API must bind to localhost only by default.

## 6) Plugin rules
- Plugins are external processes discovered via `plugin.yaml|json`.
- JSON-RPC over stdio is the v1 transport.
- Plugin manifests must declare protocol_version = 1 and capability names.

## 7) Skill registry (public subset)
These skills are vendored under `skills/` for public use in this repo:
- `core-ask-questions-if-underspecified`
- `core-qna-synthesis`
- `core-holistic-planning-and-tracking`
- `eng-go-cli-developer`
- `eng-go-developer`
- `eng-code-review-playbook`
- `eng-smart-test-runner`
- `eng-conventional-commit-helper`
- `eng-plugin-authoring`
- `eng-ui-vite-react`

If a task needs a new skill, add a public-safe version under `skills/` and update this list.
