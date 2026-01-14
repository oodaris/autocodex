# Repository Guidelines

Concise contributor guide for Autocodex. Full workflow: `docs/AGENTS.md`.

## Project overview
- **CLI + API**: Go.
- **UI**: React + Vite (deployable to Vercel).
- **Plugins**: External process plugins with manifest + RPC.
- **State**: Repo-local markdown memory and JSONL logs.

## Project structure (planned)
- `cmd/autorunner`: CLI entrypoint.
- `internal/orchestrator`: loop engine + scheduling.
- `internal/skills`: skill loader/selector.
- `internal/plugins`: plugin protocol + runner.
- `internal/state`: memory docs + logs.
- `web/`: React/Vite UI.
- `docs/`: plans, runbooks, design notes.

## Build & test (planned)
- Go tests: `go test ./...`
- Go lint: `golangci-lint run ./...`
- UI dev/build: `cd web && npm i && npm run dev|build`

## Engineering principles
- Typed I/O only. No `any` in TS.
- Contract-first: update schemas/specs before code.
- Small diffs, one behavior change per PR.
- Observability baked in (JSON logs + metrics).
- Feature flags and safe rollouts for risky paths.

## Documentation
- Plans live in `docs/plans/`.
- Add/adjust docs with each behavior change.
