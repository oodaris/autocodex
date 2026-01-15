# Autocodex v1 Plan

## Problem statement
Create an open-source Codex runner that can autonomously execute a skill-driven workflow: ideate → plan → implement → review → test → iterate. The system should start with a CLI and local API, then add a React/Vite UI (Vercel deployable). It should use Beads for task tracking, default to a configurable “yolo mode,” and support dynamically loaded plugins.

## Success criteria
- CLI can run a full loop using local Codex CLI with skill-scoped prompts.
- Beads-based tasks are created and progressed without manual `.beads/` edits.
- Config schema and API contract exist before implementation (contract-first).
- Structured logs include `trace_id, route, status, latency_ms` plus bead/task IDs.
- Local API exposes status, events, and run history; UI consumes it.
- UI loads locally via Vite and deploys to Vercel.
- At least one sample plugin works via external process RPC.

## Scope
### In scope
- Go CLI + orchestration loop.
- Local HTTP API (status, runs, events, artifacts).
- Plugin protocol (external process) + manifest + sample plugin.
- Repo-local memory docs + JSONL logs.
- React/Vite UI (read-only dashboard initially).
- Tests: unit + integration; smoke e2e for API/UI.
- Release artifacts: cross-platform CLI binaries.

### Out of scope (v1)
- Multi-agent parallel orchestration.
- Cloud-hosted Codex execution.
- Jira sync or Tempo logging.
- Production-grade auth/multi-tenant controls.

## Constraints
- Language/runtime: Go (CLI + API).
- UI: React + Vite (deployable to Vercel).
- Execution mode: local Codex CLI only.
- Beads required; no Jira.
- Default mode: “yolo” with explicit config and warnings.

## Architecture overview
- **CLI**: Cobra-based command tree.
- **Orchestrator**: state machine driving phases (ideate → plan → execute → review → test → iterate).
- **Skills**: loaded from repo or global skill paths; prompt assembly with allowlisted sections.
- **State**: repo-local markdown memory and JSONL events.
- **Plugins**: external process protocol, versioned handshake.
- **API**: local HTTP server to expose status/events and support the UI.

### Plugin recommendation
Use **external process plugins** with a manifest + gRPC (HashiCorp go-plugin) or JSON-RPC. This avoids Go’s native `plugin` limitations and supports cross-platform + multi-language plugins.

## Phases & dependencies

### Phase 0 — Repo bootstrap
**Goal**: establish docs + Beads workflow.
- Create docs/AGENTS and Repo-Guidelines.
- Initialize `.beads/` with `bd init`.
- Create plan and BD tasks.

**Acceptance**: repo has planning + Beads scaffolding; bd lint passes.

---

### Phase 1 — Contracts (must precede code)
**Goal**: define contracts/specs for config, API, and plugin protocol.
- Config schema (YAML + JSON Schema). Include `mode: yolo|safe` and Codex CLI path.
- API contract (OpenAPI): status, runs, events, artifacts.
- Plugin manifest + protocol spec (handshake, capabilities, I/O).

**Acceptance**:
- Schema docs and OpenAPI exist in `docs/contracts/`.
- Example config in `.env.example` + `config.example.yaml`.

**Dependencies**: Phase 0.

---

### Phase 2 — CLI Orchestrator core
**Goal**: implement the loop engine with state + memory docs.
- CLI commands: `init`, `run`, `status`, `beads`, `config`.
- Loop engine that executes phases using local Codex CLI.
- State store (JSONL events, run history, memory docs).

**Acceptance**:
- `autocodex run` produces logs + memory docs.
- Loop respects config and exits cleanly.

**Dependencies**: Phase 1.

---

### Phase 3 — Plugin system
**Goal**: implement plugin runner + sample plugin.
- Plugin protocol implementation (external process RPC).
- Manifest discovery (`plugins/` + PATH).
- Sample plugin: “task summarizer” or “log analyzer”.

**Acceptance**:
- CLI can load and call a plugin.
- Plugins are versioned + handshake validated.

**Dependencies**: Phase 1, Phase 2.

---

### Phase 4 — Local API
**Goal**: expose run data for UI + external tools.
- HTTP server with endpoints defined by OpenAPI.
- Read-only for v1 (no writes beyond start/stop/run).

**Acceptance**:
- OpenAPI endpoints implemented + tested.
- API serves local run/event data.

**Dependencies**: Phase 1, Phase 2.

---

### Phase 5 — UI (React/Vite)
**Goal**: local dashboard + Vercel deploy.
- Vite app reads from local API.
- Views: runs list, run detail, event stream, memory docs.

**Acceptance**:
- `npm run dev` UI renders with local API.
- `npm run build` succeeds; Vercel deploy docs included.

**Dependencies**: Phase 4.

---

### Phase 6 — Tests + CI
**Goal**: add quality gates and reproducible checks.
- Go unit tests and integration tests for CLI + API.
- UI tests (smoke + component tests).
- CI workflow: go test + lint + UI build.

**Acceptance**:
- CI passes on clean checkout.
- Coverage ≥90% on changed files (where feasible).

**Dependencies**: Phases 2–5.

---

### Phase 7 — Docs + Release
**Goal**: prepare OSS release.
- README, quickstart, config reference.
- Release build (goreleaser) and versioning.
- Rollout/rollback notes (for releases).

**Acceptance**:
- README + docs explain setup and safe mode.
- Release artifacts build locally.

**Dependencies**: Phases 2–6.

## Risks & mitigations
- **Plugin ABI drift** → versioned protocol + handshake validation.
- **Yolo mode safety** → explicit warnings + safe mode docs.
- **Codex CLI changes** → pin CLI version in docs; detect version at runtime.
- **Local API security** → bind to localhost only by default.

## Evidence checklist
- Config schema + OpenAPI spec committed.
- Test outputs (go test, UI build/test).
- Logs or screenshots for CLI + UI.
- Release artifacts build logs.

## Rollout/rollback
- Release via git tag + goreleaser.
- Rollback by reverting tag and republishing.

## Next 10 work items (v1 follow-ups)
Ordered to respect the golden workflow and current dependencies.

1. **autocodex-tju** — Contracts: memory docs API endpoints.
2. **autocodex-ek0** — API: implement memory docs endpoints (depends on autocodex-tju).
3. **autocodex-hx2** — UI: memory docs view (depends on autocodex-ek0).
4. **autocodex-704** — Docs: UI usage + Vercel deploy notes (depends on autocodex-hx2).
5. **autocodex-0bm** — UI dashboard: close after docs (now depends on autocodex-704).
6. **autocodex-41y** — UI: add polling/auto-refresh (depends on autocodex-704).
7. **autocodex-12q** — Integration smoke: autocodex run with Codex CLI.
8. **autocodex-cdu** — Tooling: add a11y + bundlesize scripts (depends on autocodex-0bm).
9. **autocodex-llw** — Tests + CI gates.
10. **autocodex-pzy** — Docs + release packaging.
