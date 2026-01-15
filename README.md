![autocodex banner](docs/assets/autocodex-banner.png)

# autocodex

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

## Quickstart
1) Install Go and the Codex CLI.
2) Copy the example config:
```bash
cp config.example.yaml autocodex.yaml
```
3) Run the CLI:
```bash
go run ./cmd/autocodex init --config autocodex.yaml
go run ./cmd/autocodex run --config autocodex.yaml
```
4) (Optional) Start the UI:
```bash
cd web
npm install
npm run dev
```

## Configuration
- `autocodex.yaml` controls mode, paths, Codex CLI settings, plugins, and API settings.
- `mode: yolo` is explicit and must be used intentionally.
- `hub.enabled` adds multi-repo workspace tracking.
- `auth.enabled` enforces API tokens (see `docs/ui/README.md`).
- `auth.token_env` can read a token from an environment variable.

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
go run ./cmd/autocodex snapshot --run <run-id> --reason "handoff"
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

## License
MIT — see `LICENSE`.
