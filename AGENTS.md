# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

Repo note: this repo's `.beads` is on the Dolt backend (embedded). Verify with `bd backend show --json`.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## autocodex defaults (for agents)

Use `autocodex bootstrap` when you want autonomy ready immediately.
- Creates `autocodex.yaml` if missing (falls back to the embedded config if `config.example.yaml` is absent).
- Creates autonomy templates/schemas in `docs/`.
- Writes a minimal skill pack to `skills/` and expects `skills.paths` to include `skills`.
- Does not overwrite existing files unless `--force` is provided.
- If `bd` is missing, bead creation/updates are skipped with a warning.

Use `autocodex init` for a minimal setup (config + `.autocodex/` only).

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
