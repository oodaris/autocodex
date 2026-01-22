![autocodex banner](docs/assets/autocodex-banner.png)

# autocodex

[![ci](https://github.com/oodaris/autocodex/actions/workflows/ci.yml/badge.svg)](https://github.com/oodaris/autocodex/actions/workflows/ci.yml)
[![coverage](https://codecov.io/gh/oodaris/autocodex/branch/main/graph/badge.svg)](https://codecov.io/gh/oodaris/autocodex)
[![Go Report Card](https://goreportcard.com/badge/github.com/oodaris/autocodex)](https://goreportcard.com/report/github.com/oodaris/autocodex)
[![Scorecard](https://api.securityscorecards.dev/projects/github.com/oodaris/autocodex/badge)](https://securityscorecards.dev/viewer/?uri=github.com/oodaris/autocodex)
[![release](https://img.shields.io/github/v/release/oodaris/autocodex?display_name=tag&sort=semver)](https://github.com/oodaris/autocodex/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![go](https://img.shields.io/badge/go-1.22%2B-blue)](go.mod)

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

Or download a release binary from GitHub Releases (recommended for non-Go setups):
```bash
# requires gh (GitHub CLI)
gh release download -R oodaris/autocodex -p "autocodex_*_darwin_arm64.tar.gz"
tar -xzf autocodex_*_darwin_arm64.tar.gz
sudo mv autocodex /usr/local/bin/autocodex
```

**Latest release (version‑pinned by variable)**
```bash
# requires gh (GitHub CLI)
VERSION=$(gh release view -R oodaris/autocodex --json tagName -q .tagName | sed 's/^v//')
ARCH=darwin_arm64
gh release download "v${VERSION}" -R oodaris/autocodex -p "autocodex_${VERSION}_${ARCH}.tar.gz"
tar -xzf "autocodex_${VERSION}_${ARCH}.tar.gz"
sudo mv autocodex /usr/local/bin/autocodex
autocodex --version
```

## Quickstart
**Prereqs**
- Go 1.22+

CLI reference: `docs/CLI.md`
- Codex CLI on PATH (`codex --version`)
- Parallelism & collaboration guide: `docs/parallelism-and-collaboration.md`

**Parallelism vs collaboration (quick blurb)**  
Autocodex **parallelism** is handled by the coordinator (`--swarm`), which runs
multiple Codex CLI processes in parallel. Codex `collaboration_mode/preset`
controls **role‑style collaboration inside a single process**. Use both if you
want parallel beads **and** in‑process collaboration.

Disable collaboration for a single run:
```bash
autocodex run --no-collaboration --task "Run without collaboration presets"
```

### Choose your setup

We ship two setup commands so open‑source users can choose between a minimal
footprint and a full autonomy experience.

**Initialize** (minimal setup)
- Creates `autocodex.yaml` if missing.
- Creates local state + memory docs under `.autocodex/`.
- Initializes a git repo + Beads if missing (disable with `--init-git=false` / `--init-bd=false`).
- Best when you want to supply your own templates/skills or keep autonomy off.

```bash
autocodex init
```

**Bootstrap** (full autonomy out of the box)
- Creates `autocodex.yaml` if missing.
- Initializes a git repo + Beads if missing (disable with `--init-git=false` / `--init-bd=false`).
- Writes autonomy templates + schemas into `docs/`.
- Writes a minimal skill pack into `skills/` so autonomy can run immediately.
- Does **not** overwrite existing files unless you pass `--force`.
- If `bd` is missing, bead tracking is skipped with a warning.
- Best for new contributors or repos that want autonomy by default.

```bash
autocodex bootstrap
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

# inspect status as a table (with filters)
autocodex status --table --status failed --limit 10

# list runs (table)
autocodex runs

# force bead-parallel coordinator (swarm)
autocodex run --swarm --task "Run all unblocked beads in parallel"
# Prefer --swarm for guaranteed parallelism (uses the Autocodex coordinator)

# inspect latest status
autocodex status --latest

# resume a run using snapshot context (starts a new run)
autocodex resume --run <run-id> --task "Continue from the previous run"

# list runs and pick interactively (TTY only)
autocodex resume --list
autocodex resume

# resume without adding a new task
autocodex resume --run <run-id> --force

# start the local API
autocodex api

# start the embedded UI + API
autocodex ui

# print version
autocodex --version

# snapshot a run
autocodex snapshot 20260115T142253Z-4a4ae121 --reason "handoff"

# clean up old runs (14 days by default)
autocodex cleanup --dry-run

# remove a specific run (and its logs)
autocodex cleanup --run <run-id>
```

**UI (embedded)**
```bash
autocodex ui
```

**UI dev server (Vite)**
```bash
cd web
npm install
npm run dev
```

## Troubleshooting
- **Address already in use (API/UI won't start)**: stop the existing process listening on the port or change `api.port` in `autocodex.yaml`.
- **Auth enabled but no tokens resolved**: set `auth.tokens` in `autocodex.yaml` or provide `AUTH_TOKEN` via `auth.token_env`.
- **`bd` not installed**: install Beads (`bd`) or disable Beads features in config (`beads.enabled: false`).

**Custom config path**
```bash
autocodex run --config path/to/autocodex.yaml --task "Review backend API and fix issues."
```

**Pipe a task from stdin**
```bash
echo "Review backend API and fix issues." | autocodex run --task-stdin
```

**Resume from an existing plan/phase**
```bash
# start at implementation, using the latest spec/plan/tasks artifacts
autocodex run --start-phase implement --use-latest-artifacts \
  --task "Continue from existing artifacts"
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
 - Plans should include explicit must-have gates (tests, runtime verification, evidence paths) so autonomy can enforce completion.

By default autocodex enables Codex collaboration (`codex.collaboration_mode: auto`, `codex.preset: default`).
Override with `--collaboration-mode/--preset` or in `autocodex.yaml` if you need different behavior.

### Artifacts and cleanup
Each run stores artifacts under `.autocodex/runs/<run-id>/artifacts` and logs under `.autocodex/logs/<run-id>.jsonl`.
At the end of a run, autocodex prints a short summary (run id + spec/plan/tasks paths) and a cleanup command you can copy:
```bash
autocodex cleanup --run <run-id>
```
Use `autocodex cleanup --dry-run` (or `--retention-days`) to prune older runs safely.

### Autonomy checklist
- `autocodex.yaml` exists and `autonomy.enabled: true`.
- Templates + schemas exist (run `autocodex bootstrap` to create them): `docs/specs/TEMPLATE.md`, `docs/plans/TEMPLATE.md`, `docs/contracts/autonomy-tasks.schema.json`, `docs/contracts/autonomy-actions.schema.json`.
- Skills available in `skills/`: `core-qna-synthesis`, `core-holistic-planning-and-tracking`, `core-ask-questions-if-underspecified`.

### Parallel coordinator (optional)
You can run beads in parallel with the autonomy coordinator:
```yaml
autonomy:
  coordinator:
    enabled: true
    max_parallel: 2   # 0 = unlimited
    strategy: bead    # or "phase" for isolated phases
    fail_fast: false  # stop all beads on first error
```
Notes:
- Parallel mode ignores `require_next`.
- Memory docs are isolated per bead under `memory/beads/<id>`.
- Codex CLI installed and reachable (`codex` on PATH or `codex.cli_path`).
- `bd` is optional; without it, bead creation/updates are skipped with a warning.

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
`autocodex ui` serves the embedded production UI and starts the API server.
See `docs/ui/README.md` for local dev + Vercel deployment notes.

Building from source requires a UI build before Go compile:
```bash
npm ci --prefix web
npm run build --prefix web
```

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
