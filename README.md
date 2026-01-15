![autocodex banner](docs/assets/autocodex-banner.png)

# autocodex

![ci](https://github.com/oodaris/autocodex/actions/workflows/ci.yml/badge.svg)

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
Option A: install with Go:
```bash
go install github.com/oodaris/autocodex/cmd/autocodex@latest
```

Option B: download a release binary from GitHub Releases.

## Quickstart
**Prereqs**
1) Install Go (1.22+).
2) Install the Codex CLI (ensure `codex` is on PATH).

**Install autocodex**
```bash
go install github.com/oodaris/autocodex/cmd/autocodex@latest
```

**Initialize (creates `autocodex.yaml` if missing)**
```bash
autocodex init
```

**Run a task (shortest command)**
```bash
autocodex "Review backend API and fix issues."
```

**Other common commands**
```bash
# bounded loop
autocodex once "Run a quick UI a11y review."

# snapshot a run
autocodex snapshot 20260115T142253Z-4a4ae121 --reason "handoff"
```

**Optional UI**
```bash
cd web
npm install
npm run dev
```

For a longer walkthrough, see `docs/quickstart.md`.

## Configuration
- `autocodex.yaml` controls mode, paths, Codex CLI settings, plugins, and API settings.
- `mode: yolo` is explicit and must be used intentionally.
- Default Codex model/effort: `gpt-5.2-codex` + `xhigh` (override in `autocodex.yaml` if needed).
- `hub.enabled` adds multi-repo workspace tracking.
- `auth.enabled` enforces API tokens (see `docs/ui/README.md`).
- `auth.token_env` can read a token from an environment variable.
For a full reference, see `docs/config/README.md`.

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
