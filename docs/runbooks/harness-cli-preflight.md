# Harness CLI Preflight

Run deterministic checks before high-impact autonomy execution or harness policy edits.

## Command
```bash
bash scripts/dev/harness-cli-preflight.sh
```

Optional strict mode:
```bash
autocodex harness preflight --strict
```

## Checks
1. `bd` command availability and initialized repo state.
2. `bd --version` meets this repo baseline (`>=0.56.1`).
3. Dolt readiness (`bd dolt show --json`) via doctor checks.
4. `codex` CLI availability and version/capability checks.
5. `autocodex harness preflight --strict` (or go-run fallback), which includes doctor + harness lint checks.
6. Standalone harness config lint (`python3 scripts/harness_config_lint.py`) as explicit policy-pack validation.

## Success marker
`Harness preflight passed.`

## Troubleshooting
1. If `bd` is uninitialized in this clone, run:
   - `bd init --from-jsonl`
2. If `bd --version` is below `0.56.1`, upgrade before strict preflight:
   - `go install github.com/steveyegge/beads/cmd/bd@v0.56.1`
   - or `npm i -g @beads/bd`
3. If Dolt readiness fails, run:
   - `bd dolt show --json`
   - `bd doctor --migration=post`
   - if `connection_ok` is false, start Dolt SQL server for this repo:
     - `dolt sql-server --data-dir "$(pwd)/.beads/dolt" --host 127.0.0.1 --port 3307`
4. If lint fails, resolve missing role/config/doc markers.
5. If doctor fails feature checks, align Codex CLI and config assumptions.
