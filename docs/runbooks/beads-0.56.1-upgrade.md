# Beads 0.56.1 Upgrade Runbook

Use this runbook to align local/operator environments with the Beads baseline expected by `autocodex` doctor and harness preflight.

## Scope
- Target: Beads (`bd`) `>=0.56.1`
- Backend: Dolt (`.beads/dolt`, embedded mode by default in this repo)
- Outcome: strict readiness checks pass without Beads-version or Dolt-readiness warnings

## Phase 0: Install or upgrade `bd`
Choose one install path and verify `bd` resolves from your active `PATH`.

```bash
# Go install (pin exact version)
GOBIN="$HOME/.local/bin" go install github.com/steveyegge/beads/cmd/bd@v0.56.1

# npm global
npm i -g @beads/bd

# Homebrew
brew install steveyegge/beads/bd
```

Verify:

```bash
bd --version
command -v bd
```

## Phase 1: Validate Dolt + repo readiness
Run from repo root:

```bash
bd info --json
bd dolt show --json
bd doctor --migration=post
```

If `bd dolt show --json` reports `"connection_ok": false`, start a Dolt SQL server for this repo:

```bash
dolt sql-server --data-dir "$(pwd)/.beads/dolt" --host 127.0.0.1 --port 3307
```

Then re-run:

```bash
bd dolt test --json
bd info --json
```

Expected:
- `bd info --json` reports `database_path` under `.beads/dolt`.
- `bd dolt show --json` reports `"backend": "dolt"` and `connection_ok: true`.
- `bd doctor --migration=post` returns healthy migration/readiness status.

## Phase 2: Verify autocodex strict gates
Run strict readiness checks:

```bash
autocodex doctor --config config.example.yaml --strict
autocodex harness preflight --config config.example.yaml --strict
python3 scripts/harness_config_lint.py
```

Success criteria:
- No `doctor.bd-version` warning (requires `>=0.56.1`).
- No `doctor.bd-dolt` warning for the configured mode.
- Harness preflight exits successfully.

## Notes on `bd sync`
- `bd sync` is deprecated/no-op in this setup; do not rely on it for data movement.
- Canonical state is Dolt (`.beads/dolt`), not `.beads/issues.jsonl`.
- If your team still needs JSONL mirror files, install and enforce hooks:
```bash
bd hooks install
bd hooks list --json
ENFORCE_JSONL_HOOKS=1 bash scripts/dev/harness-cli-preflight.sh
```
- Use `bd dolt pull` / `bd dolt push` only when a Dolt remote is configured.

## Rollback
If you need to revert to an older local Beads version for troubleshooting:

```bash
# example (replace with the required previous version)
GOBIN="$HOME/.local/bin" go install github.com/steveyegge/beads/cmd/bd@v0.55.4
```

Re-run Phase 1 and Phase 2 to confirm behavior and document any strict-check regressions before merging.
