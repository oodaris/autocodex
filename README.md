![autocodex banner](docs/assets/autocodex-banner.png)

# autocodex

[![ci](https://github.com/oodaris/autocodex/actions/workflows/ci.yml/badge.svg)](https://github.com/oodaris/autocodex/actions/workflows/ci.yml)
[![release](https://img.shields.io/badge/release-v0.1.0-blue)](https://github.com/oodaris/autocodex/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![go](https://img.shields.io/badge/go-1.22%2B-blue)](go.mod)
[![Go Report Card](https://img.shields.io/badge/go%20report-n%2Fa-lightgrey)](https://goreportcard.com/report/github.com/oodaris/autocodex)

autocodex orchestrates a structured loop: ideate → plan → implement → review → test. It uses Beads for task tracking, runs the local Codex CLI, and supports external plugins via JSON‑RPC.

## Features
- Go CLI with a deterministic, scriptable workflow
- Beads-first task tracking
- Local API for runs, events, artifacts, and memory docs
- External plugin system (JSON‑RPC over stdio)
- React/Vite UI for runs, events, artifacts, and memory docs
- Optional UI auto‑refresh with backoff
- Hub mode for multi-repo dashboards
- Terminal sessions (websocket PTY)
- Optional token auth for the API/UI

## Install
Install with Go:
```bash
go install github.com/oodaris/autocodex/cmd/autocodex@latest
```

Or download a release binary from GitHub Releases (recommended for non-Go setups).

## Quickstart
**Prereqs**
- Go 1.22+
- Codex CLI on PATH (`codex --version`)

**Initialize** (creates `autocodex.yaml` if missing)
```bash
autocodex init
```

**Run a task** (shortest command)
```bash
autocodex "Review backend API and fix issues."
```

**Common commands**
```bash
# bounded loop
autocodex once "Run a quick UI a11y review."

# inspect status
autocodex status

# inspect latest status
autocodex status --latest

# start the local API
autocodex api

# snapshot a run
autocodex snapshot 20260115T142253Z-4a4ae121 --reason "handoff"
```

**Optional UI**
```bash
cd web
npm install
npm run dev
```

**Custom config path**
```bash
autocodex run --config path/to/autocodex.yaml --task "Review backend API and fix issues."
```

**Pipe a task from stdin**
```bash
echo "Review backend API and fix issues." | autocodex run --task-stdin
```

For a longer walkthrough, see `docs/quickstart.md`.

## Autonomy loop
Autonomy mode generates spec/plan artifacts, creates beads, and advances through beads automatically.

Enable it in `autocodex.yaml`:
```yaml
autonomy:
  enabled: true
```

When autonomy is enabled:
- `autocodex "task"` creates spec/plan/tasks artifacts and beads.
- Beads are selected in dependency order (`bd ready`).
- The **test** phase should emit an `ACTIONS` JSON block (see `docs/contracts/autonomy-actions.schema.json`) so autocodex can update bead status and choose the next bead.
- Gate failures stop the loop and auto-create a fix bead (when beads auto-create is enabled).

## Configuration
- `autocodex.yaml` controls mode, paths, Codex CLI settings, plugins, and API settings.
- `mode: yolo` is explicit and must be used intentionally.
- Default Codex model/effort: `gpt-5.2-codex` + `xhigh` (override in `autocodex.yaml` if needed).
- `hub.enabled` adds multi-repo workspace tracking.
- `auth.enabled` enforces API tokens (see `docs/ui/README.md`).
- `auth.token_env` can read a token from an environment variable.
- `autonomy.enabled` generates spec + plan artifacts before the loop runs.
- Templates: `docs/specs/TEMPLATE.md` and `docs/plans/TEMPLATE.md`.
If the Codex CLI is not on PATH, set `codex.cli_path` in `autocodex.yaml`.
For a full reference, see `docs/config/README.md`.

## Agent requirements
To run autocodex in an agent environment:
- `autocodex.yaml` present in the repo (or pass `--config`)
- Codex CLI installed and reachable (`codex.cli_path` if not on PATH)
- Skills path configured (`skills.paths`)
- Optional: Beads (`bd`) if you want task tracking

## UI usage
See `docs/ui/README.md` for local usage and Vercel deployment notes.

## Plugins
Plugins are external processes described by `plugin.yaml`. A sample plugin lives in `plugins/sample-summarizer/`.

Build the sample plugin:
```bash
go build -o plugins/sample-summarizer/sample-summarizer ./plugins/sample-summarizer
```

List plugins:
```bash
go run ./cmd/autocodex plugins --action list
```

Run the sample plugin:
```bash
go run ./cmd/autocodex plugins --action run \
  --name sample-summarizer \
  --capability summarize \
  --input '{"text":"hello world"}'
```

## Snapshots
Generate a run snapshot (memory docs + recent events/artifacts) for sharing or continuity:
```bash
autocodex snapshot <run-id> --reason "handoff"
```

## Troubleshooting
- `codex` not found: set `codex.cli_path` in `autocodex.yaml`.
- API 401: set `auth.enabled: true` and provide `auth.token_env` or `auth.tokens`.
- API 404 at `/`: ensure `api.base_path` is `/` and hit `/health` for a quick check.
- UI shows zero runs: confirm `autocodex api` is running and `ui.origin` matches the UI URL.
- Hub shows “hub not enabled”: set `hub.enabled: true` (or add workspaces in `autocodex.yaml`).

## Development
- Tests: `go test ./...`
- Vet: `go vet ./...`
- Format: `gofmt -w $(rg --files -g '*.go')`

## Repo Guide
- Agent instructions: `AGENTS.md`
- Engineering playbook: `docs/AGENTS.md`
- Plan: `docs/plans/autocodex-v1-plan.md`
- Contracts: `docs/contracts/`
- Plugins guide: `docs/plugins/README.md`
- UI guide: `docs/ui/README.md`

## Release
- Changelog: `CHANGELOG.md`
- GoReleaser config: `goreleaser.yml`
- Release process: `docs/release/README.md`

## License
MIT — see `LICENSE`.
