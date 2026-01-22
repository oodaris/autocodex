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

> 🚀 **Quickstart**
> 1) `autocodex bootstrap`  
> 2) `autocodex "Review backend API and fix issues."`  
> 3) `autocodex ui`

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

### Option A — Go (fastest)
```bash
go install github.com/oodaris/autocodex/cmd/autocodex@latest
```

### Option B — GitHub release binary
```bash
# requires gh (GitHub CLI)
gh release download -R oodaris/autocodex -p "autocodex_*_darwin_arm64.tar.gz"
tar -xzf autocodex_*_darwin_arm64.tar.gz
sudo mv autocodex /usr/local/bin/autocodex
```

> 💡 **Tip**
> Want a version‑pinned install? See `docs/quickstart.md` for the latest‑version
> install snippet (auto‑detects the newest tag).

## Quickstart

**Prereqs**
- Go 1.22+
- Codex CLI on PATH (`codex --version`)

**Docs**
- CLI reference: `docs/CLI.md`
- Parallelism & collaboration: `docs/parallelism-and-collaboration.md`

### Choose your setup

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

```bash
autocodex bootstrap
```

### Run a task
```bash
autocodex "Review backend API and fix issues."
```

### Start the UI
```bash
autocodex ui
```

> ⚡ **Parallelism vs collaboration**
> `--swarm` uses the autocodex coordinator to run **multiple Codex processes** in parallel.
> Codex `collaboration_mode/preset` controls **role‑style collaboration inside a single process**.
> Use both if you want parallel beads **and** in‑process collaboration.

Disable collaboration for a single run:
```bash
autocodex run --no-collaboration --task "Run without collaboration presets"
```

## Parallelism & collaboration
- **Guaranteed parallelism**: `autocodex run --swarm`  
- **Role‑style collaboration**: `--collaboration-mode/--preset`  
Full guide: `docs/parallelism-and-collaboration.md`

## CLI cheat sheet
<details>
<summary>Common commands</summary>

```bash
# bounded loop
autocodex once "Run a quick UI a11y review."

# inspect status
autocodex status
autocodex status --table --status failed --limit 10
autocodex runs

# parallel beads
autocodex run --swarm --task "Run all unblocked beads in parallel"

# resume a run
autocodex resume --run <run-id> --task "Continue from the previous run"
autocodex resume --list
autocodex resume --run <run-id> --force

# api + ui
autocodex api
autocodex ui

# snapshot + cleanup
autocodex snapshot <run-id> --reason "handoff"
autocodex cleanup --dry-run
autocodex cleanup --run <run-id>
```
</details>

## Autonomy loop
<details>
<summary>How autonomy works</summary>

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
- Plans should include explicit must-have gates (tests, runtime verification, evidence paths).

Parallel coordinator (optional):
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
- Codex CLI must be installed and reachable (`codex` on PATH or `codex.cli_path`).
- `bd` is optional; without it, bead creation/updates are skipped with a warning.
</details>

## Configuration
autocodex uses `autocodex.yaml` for all runtime settings.  
See `docs/config/README.md` for the full reference.

## Troubleshooting
- **Address already in use** (API/UI won’t start): stop the existing process or change `api.port`.
- **`codex` not found**: set `codex.cli_path` in `autocodex.yaml`.
- **API 401**: set `auth.enabled: true` and provide `auth.token_env` or `auth.tokens`.
- **API 404 at `/`**: ensure `api.base_path` is `/` and hit `/health`.
- **UI shows zero runs**: confirm `autocodex api` is running and `ui.origin` matches the UI URL.
- **Hub not enabled**: set `hub.enabled: true` (or add workspaces in `autocodex.yaml`).
- **`bd` not installed**: install Beads (`bd`) or disable in config (`beads.enabled: false`).

## Development
- Tests: `go test ./...`
- Vet: `go vet ./...`
- Format: `gofmt -w $(rg --files -g '*.go')`

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

## Repo guide
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
