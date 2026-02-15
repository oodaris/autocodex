![autocodex banner](docs/assets/autocodex-banner.png)

# autocodex

[![ci](https://github.com/oodaris/autocodex/actions/workflows/ci.yml/badge.svg)](https://github.com/oodaris/autocodex/actions/workflows/ci.yml) [![scorecard](https://github.com/oodaris/autocodex/actions/workflows/scorecard.yml/badge.svg)](https://github.com/oodaris/autocodex/actions/workflows/scorecard.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/oodaris/autocodex)](https://goreportcard.com/report/github.com/oodaris/autocodex) [![release](https://img.shields.io/github/v/release/oodaris/autocodex?display_name=tag&sort=semver)](https://github.com/oodaris/autocodex/releases/latest) [![license](https://img.shields.io/badge/license-MIT-blue)](LICENSE) [![go](https://img.shields.io/badge/go-1.26%2B-blue)](go.mod)

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

### Option B — GitHub release binary (one‑liner)
```bash
curl -fsSL https://raw.githubusercontent.com/oodaris/autocodex/main/scripts/install.sh | bash
```

> 💡 **Tip**
> Pin a version or install to a custom path:
> ```bash
> VERSION=0.5.1 DEST=~/.local/bin \
>   curl -fsSL https://raw.githubusercontent.com/oodaris/autocodex/main/scripts/install.sh | bash
> ```

### Option C — GitHub release binary (manual)
```bash
# requires gh (GitHub CLI)
gh release download -R oodaris/autocodex -p "autocodex_*_darwin_arm64.tar.gz"
tar -xzf autocodex_*_darwin_arm64.tar.gz
sudo mv autocodex /usr/local/bin/autocodex
```

> 💡 **Tip**
> Want a version‑pinned install? See `docs/quickstart.md` for the latest‑version
> install snippet (auto‑detects the newest tag).

Verify install:
```bash
autocodex --version
autocodex plugins --action list
```

## Quickstart

**Runtime requirements (binary users)**
- Codex CLI on PATH (`codex --version`)
  - Tested with: `codex-cli 0.101.0`
  - Recommended: `codex-cli >= 0.94.0` (older versions may work but have different defaults around approvals/search/modes)
- Beads (`bd`) optional, required when `autonomy.require_bd: true`

**Build / contributor requirements**
- Go 1.26+ (only if installing via `go install` or building from source)
- Node.js + npm (only for UI dev)

**Install Codex CLI** (GitHub: `openai/codex`)
```bash
# npm
npm i -g @openai/codex
# or homebrew
brew install --cask codex
```

**Install Beads (`bd`)** (GitHub: `steveyegge/beads`)
```bash
# go
go install github.com/steveyegge/beads/cmd/bd@latest
# or homebrew
brew install steveyegge/beads/bd
```

**Docs**
- CLI reference: `docs/CLI.md`
- Parallelism & collaboration: `docs/parallelism-and-collaboration.md`

### What happens on run
```
task → spec → plan → tasks.json → beads → loop (ideate/plan/implement/review/test)
```

### Autonomy vs run
| Mode | What it does | When to use |
| --- | --- | --- |
| `run` / `once` | Executes the loop with your task | Quick iteration or manual control |
| `autonomy` | Generates spec/plan/tasks + beads and advances automatically | Larger tasks with dependencies |

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

### Cost control / performance
Use these settings to reduce compute:
- `codex.reasoning_effort`: `low` / `medium` / `high` (avoid `xhigh` if speed/cost matters)
- `autonomy.coordinator.max_parallel`: limit concurrent bead runs
- `autonomy.stop_conditions.max_fix_attempts` and `max_beads`: cap retries and total beads

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

## Flag index (top‑level)
| Flag | Commands | Purpose |
| --- | --- | --- |
| `--config` | most | config file path |
| `--task` / `--task-file` / `--task-stdin` | run/once/resume | task input |
| `--start-phase` | run/once/resume | start at a specific phase |
| `--use-latest-artifacts` | run/once/resume | reuse latest spec/plan |
| `--swarm` | run/once/resume | force coordinator (parallel beads) |
| `--no-collaboration` | run/once/resume | disable collaboration defaults |
| `--collaboration-mode` / `--preset` | run/once/resume | collaboration settings |
| `--run` | resume/snapshot/kill/cleanup | run id |
| `--list` | resume | list runs and exit |
| `--force` | resume | resume even if completed/running |
| `--json` | status/snapshot/cleanup/kill | machine‑readable output |
| `--dry-run` | cleanup | preview deletions |

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
### If your first run fails
```bash
codex --version
autocodex doctor
autocodex status --latest
```

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
Plugins are external processes described by `plugin.yaml`. See the catalog and usage details in `docs/plugins/README.md`.

### Plugin catalog
| Plugin | Capability | Purpose | Output |
| --- | --- | --- | --- |
| `repo-indexer` | `index` | Project map: languages, key dirs, test commands, services | JSON summary for models + humans |
| `test-runner` | `run` | Run scoped tests with timeouts | Pass/fail + command logs |
| `diff-summarizer` | `summarize` | Summarize git diff + risk flags | Areas + risk flags |
| `dep-license-scanner` | `scan` | Extract dependencies + license risks | Dependencies + risk flags |
| `knowledge-extractor` | `extract` | Parse docs into structured summaries | Doc list + headings/snippets |
| `plan-compliance` | `check` | Validate plan sections + open tasks | Status + missing items |
| `evidence-collector` | `collect` | Capture evidence artifacts | Artifact manifest |

### Plugin distribution
- Release archives include **prebuilt plugins + manifests**.
- The install script copies plugins to `${PREFIX}/share/autocodex/plugins`.
- Default plugin search paths include that location (see `docs/plugins/README.md`).

### Plugin recipes (multi‑plugin workflows)
- **Repo onboarding**: repo‑indexer → knowledge‑extractor
- **PR triage**: diff‑summarizer → dep‑license‑scanner → test‑runner
- **Plan compliance**: plan‑compliance → evidence‑collector
Full examples: `docs/plugins/README.md`

Build the example plugin:
```bash
go build -o plugins/example-summarizer/example-summarizer ./plugins/example-summarizer
```

List plugins:
```bash
go run ./cmd/autocodex plugins --action list
```

Run the example plugin:
```bash
go run ./cmd/autocodex plugins --action run \
  --name example-summarizer \
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
