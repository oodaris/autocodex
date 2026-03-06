# Harness CLI Preflight

Run deterministic checks before high-impact autonomy execution or harness policy edits.

## Command
```bash
bash scripts/dev/harness-cli-preflight.sh
```

By default, the script attempts to auto-start a local Dolt SQL server for this repo when unreachable. Disable with `AUTO_START_DOLT_SERVER=0`.

Optional strict mode:
```bash
autocodex harness preflight --strict
```

In a source checkout, bare harness commands automatically use repo-root
`config.example.yaml` when `autocodex.yaml` is absent and neither `--config`
nor `AUTOCODEX_CONFIG` is set.

## Checks
1. `bd` command availability and repo-state probe (`bd info --json` from repo root).
2. `bd --version` meets this repo baseline (`>=0.56.1`).
3. Dolt readiness (`bd dolt show --json`) via doctor checks.
4. Optional hook audit (`bd hooks list --json`) to detect JSONL mirror drift risk.
5. `codex` CLI availability and version/capability checks.
6. `autocodex harness preflight --strict` (or go-run fallback), which includes doctor + harness lint checks.
7. Standalone harness config lint (`autocodex harness lint` or go-run fallback) as explicit policy-pack validation.
8. Profile invariants: root and `max_capability` must stay on `gpt-5.4`, while Spark must keep `model_reasoning_summary = "none"`.

## Success marker
`Harness preflight passed.`

## Troubleshooting
1. If `bd` is uninitialized in this clone, run:
   - `bd onboard`
   - optional mirror setup: `bd migrate sync beads-sync`
2. If `bd --version` is below `0.56.1`, upgrade before strict preflight:
   - `go install github.com/steveyegge/beads/cmd/bd@v0.56.1`
   - or `npm i -g @beads/bd`
3. If Dolt readiness fails, run:
   - `bd dolt show --json`
   - `bd doctor --migration=post`
   - if `connection_ok` is false and auto-start is disabled/unavailable, start Dolt SQL server for this repo:
     - `dolt sql-server --data-dir "$(pwd)/.beads/dolt" --host 127.0.0.1 --port 3307`
4. If hooks are missing and your workflow requires JSONL mirror files:
   - `bd hooks install`
   - `ENFORCE_JSONL_HOOKS=1 bash scripts/dev/harness-cli-preflight.sh`
5. If lint fails, resolve missing role/config/doc markers.
6. If doctor fails feature checks, align Codex CLI and config assumptions.
7. If lint reports model-policy drift, restore root/`max_capability`/role-pack defaults to `gpt-5.4` and keep Spark isolated from unsupported `reasoning.summary` settings.
