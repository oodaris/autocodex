# autocodex CLI Reference

This document lists all commands and flags in one place. All commands accept `--help` for short usage.

## Quick start
```bash
autocodex "Review backend API and fix issues."
```

## Global behavior
- Default config path: `autocodex.yaml` (override with `AUTOCODEX_CONFIG` or `--config`).
- Shortcut: `autocodex "<task>"` is equivalent to `autocodex run --task "<task>"`.

## Commands

### bootstrap
Initialize a repo with config, templates/schemas, a minimal skill pack, and (optionally) git + beads.
```bash
autocodex bootstrap [--config <path>] [--force] [--init-git] [--init-bd]
```
Flags:
- `--config`: config file path (default `autocodex.yaml`)
- `--force`: overwrite existing templates/schemas/skills
- `--init-git`: initialize a git repo if missing (default: true)
- `--init-bd`: initialize beads if missing (default: true)

### init
Minimal setup (config + `.autocodex/`).
```bash
autocodex init [--config <path>] [--init-git] [--init-bd]
```
Flags:
- `--config`: config file path
- `--init-git`: initialize a git repo if missing (default: true)
- `--init-bd`: initialize beads if missing (default: true)

### run
Run one loop (bounded or continuous based on config).
```bash
autocodex run [--config <path>] [--task <text> | --task-file <path> | --task-stdin] [--start-phase <phase>] [--use-latest-artifacts]
```
Flags:
- `--config`: config file path
- `--task`: task text (also appended to `memory/TODO.md`)
- `--task-file`: read task text from file
- `--task-stdin`: read task text from stdin
- `--start-phase`: start at a specific phase (ideate/plan/implement/review/test)
- `--use-latest-artifacts`: when starting after ideate, append latest spec/plan paths to the task (default true)

### once
Run a single bounded loop (one pass through phases).
```bash
autocodex once [--config <path>] [--task <text> | --task-file <path> | --task-stdin] [--start-phase <phase>] [--use-latest-artifacts]
```
Flags: same as `run`.

### resume
Resume from a previous run using snapshot context (starts a new run).
```bash
autocodex resume [--config <path>] [--run <run-id>] [--task <text> | --task-file <path> | --task-stdin] [--start-phase <phase>] [--use-latest-artifacts] [--force]
```
Flags:
- `--config`: config file path
- `--run`: run id to resume from (optional if using `--list` or interactive selection)
- `--task`: optional task text to append before resume
- `--task-file`: task text from file
- `--task-stdin`: task text from stdin
- `--start-phase`: start at a specific phase (ideate/plan/implement/review/test)
- `--use-latest-artifacts`: when starting after ideate, append latest spec/plan paths to the task (default true)
- `--force`: resume even if run is still running, or to resume a completed run
- `--list`: list runs and exit (TTY selection if run id not provided)

### doctor
Run preflight checks for the current repo.
```bash
autocodex doctor [--config <path>] [--strict]
```
Flags:
- `--config`: config file path
- `--strict`: treat warnings as errors

### status
Show run status.
```bash
autocodex status [--config <path>] [--run <run-id>] [--latest] [--json] [--table] [--status <csv>] [--limit N]
```
Flags:
- `--config`: config file path
- `--run`: run id (optional)
- `--latest`: show latest run only
- `--json`: output JSON
- `--table`: output a table with headers
- `--status`: filter by status (csv: `running,completed,failed,canceled`)
- `--limit`: limit number of runs

### runs
Alias for `status --table`.
```bash
autocodex runs [--config <path>] [--status <csv>] [--limit N]
```

### snapshot
Create a snapshot of a run.
```bash
autocodex snapshot [--config <path>] [--run <run-id>] [--reason <text>] [--sources <csv>] [--max-bytes N] [--max-events N] [--max-artifacts N] [--memory-glob <glob>] [--json]
```
Flags:
- `--config`: config file path
- `--run`: run id (defaults to latest)
- `--reason`: snapshot reason
- `--sources`: comma-separated sources (`memory,events,artifacts`)
- `--max-bytes`: max snapshot bytes (0 = no limit)
- `--max-events`: max events (0 = no limit)
- `--max-artifacts`: max artifacts (0 = no limit)
- `--memory-glob`: filter memory docs (default `*.md`)
- `--json`: output JSON

### cleanup
Remove old runs.
```bash
autocodex cleanup [--config <path>] [--run <run-id>] [--retention-days N] [--dry-run] [--json]
```
Flags:
- `--config`: config file path
- `--run`: delete a single run by id (ignores retention-days)
- `--retention-days`: days to retain completed runs (0 = use config)
- `--dry-run`: list deletions only
- `--json`: output JSON

### kill
Request a run stop/kill.
```bash
autocodex kill [--config <path>] --run <run-id> [--reason <text>] [--json]
```
Flags:
- `--config`: config file path
- `--run`: run id
- `--reason`: optional reason
- `--json`: output JSON

### beads
Proxy for `bd` CLI.
```bash
autocodex beads --action ready
autocodex beads --action show --issue <id>
```
Flags:
- `--action`: `ready` or `show`
- `--issue`: issue id for `show`

### plugins
List or run plugins.
```bash
autocodex plugins --action list [--config <path>]
autocodex plugins --action run --name <plugin> --capability <cap> [--input <json>] [--input-file <path>] [--config <path>]
```
Flags:
- `--action`: `list` or `run`
- `--name`: plugin name (run)
- `--capability`: capability name (run)
- `--input`: JSON input string (run)
- `--input-file`: JSON input file (run)
- `--config`: config file path

### api
Start the local API.
```bash
autocodex api [--action serve] [--config <path>]
```

### ui
Start the embedded UI + API.
```bash
autocodex ui [--config <path>]
```

### config
Print the resolved config path.
```bash
autocodex config [--config <path>]
```

### version
Print the version.
```bash
autocodex version
# or
autocodex --version
```
