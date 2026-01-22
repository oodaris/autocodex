# autocodex quickstart

## Install

Option A: install with Go:
```bash
go install github.com/oodaris/autocodex/cmd/autocodex@latest
```

Option B: download a release binary from GitHub Releases.

Verify:
```bash
autocodex --help
```

Verify plugins:
```bash
autocodex plugins --action list
```

**Plugins note**  
Release archives include prebuilt plugins and the install script copies them to
`${PREFIX}/share/autocodex/plugins`. See `docs/plugins/README.md` for details.

## Configure

Copy the example config:
```bash
cp config.example.yaml autocodex.yaml
```

Edit `autocodex.yaml` for your repo paths, model, and loop settings.
See the config reference in `docs/config/README.md`.
If Codex is installed outside PATH, set `codex.cli_path`.

## Choose your setup path

We provide two setup commands so you can pick the right level of automation
for your repo.

### 1) Minimal setup (init)
Use this if you want a lightweight footprint or plan to manage autonomy assets
manually.

- Creates `autocodex.yaml` if missing.
- Creates `.autocodex/` state + memory docs.
- Leaves templates/skills untouched.

```bash
autocodex init --config autocodex.yaml
```

### 2) Full autonomy setup (bootstrap)
Use this if you want autonomy ready immediately for new contributors or
open‑source users.

- Creates `autocodex.yaml` if missing.
- Writes autonomy templates + schemas into `docs/`.
- Writes a minimal skill pack into `skills/`.
- Does not overwrite existing files unless you pass `--force`.
- If `bd` is missing, bead tracking is skipped with a warning.

```bash
autocodex bootstrap --config autocodex.yaml
```

## Run a loop

After `init` or `bootstrap`, you can run the loop:

```bash
autocodex run --config autocodex.yaml
```

**Parallelism vs collaboration (quick blurb)**  
autocodex parallelism is controlled by the coordinator (`--swarm`), which runs
multiple Codex CLI processes at once. Codex `collaboration_mode/preset` controls
role‑style behavior inside a single process. Use both if you want parallel beads
and in‑process collaboration.

Example:
```bash
autocodex run --swarm --task "Execute all ready beads in parallel"
```
Note: `--swarm` sets `autonomy.coordinator.max_parallel` to `0` (unlimited) for the run.

Disable collaboration for a specific run:
```bash
autocodex run --no-collaboration --task "Run without collaboration presets"
```

You can also append a task directly:

```bash
autocodex run --config autocodex.yaml --task "Add a memory docs summary card."
```

Pipe a task via stdin:

```bash
echo "Add a memory docs summary card." | autocodex run --task-stdin
```

Shortcut:

```bash
autocodex "Add a memory docs summary card."
```

Bounded loop shortcut:

```bash
autocodex once "Run a quick UI a11y review."
```

Snapshot shortcut:

```bash
autocodex snapshot 20260115T142253Z-4a4ae121 --reason "handoff"
```

## Start the API

```bash
autocodex api --config autocodex.yaml
```

## Start the UI (embedded)

```bash
autocodex ui --config autocodex.yaml
```

This starts the API server and serves the embedded production UI on the same port.

## Start the UI dev server (optional)

```bash
cd web
npm install
npm run dev
```
