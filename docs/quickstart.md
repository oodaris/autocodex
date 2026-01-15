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

## Configure

Copy the example config:
```bash
cp config.example.yaml autocodex.yaml
```

Edit `autocodex.yaml` for your repo paths, model, and loop settings.
See the config reference in `docs/config/README.md`.
If Codex is installed outside PATH, set `codex.cli_path`.

## Run a loop

```bash
autocodex init --config autocodex.yaml
autocodex run --config autocodex.yaml
```

You can also append a task directly:

```bash
autocodex run --config autocodex.yaml --task "Add a memory docs summary card."
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

## Start the UI (optional)

```bash
cd web
npm install
npm run dev
```
