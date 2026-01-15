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

## Run a loop

```bash
autocodex init --config autocodex.yaml
autocodex run --config autocodex.yaml
```

You can also append a task directly:

```bash
autocodex run --config autocodex.yaml --task "Add a memory docs summary card."
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
