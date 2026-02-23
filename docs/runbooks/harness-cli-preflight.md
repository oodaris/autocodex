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
2. `codex` CLI availability and version/capability checks.
3. `autocodex harness preflight --strict` (or go-run fallback), which includes doctor + harness lint checks.
4. Standalone harness config lint (`python3 scripts/harness_config_lint.py`) as explicit policy-pack validation.

## Success marker
`Harness preflight passed.`

## Troubleshooting
1. If `bd` is uninitialized in this clone, run:
   - `bd init --from-jsonl`
2. If lint fails, resolve missing role/config/doc markers.
3. If doctor fails feature checks, align Codex CLI and config assumptions.
