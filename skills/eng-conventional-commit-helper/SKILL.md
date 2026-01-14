---
name: eng-conventional-commit-helper
description: Create conventional commits with clear scope and intent.
version: 0.1.0
---

# Conventional Commit Helper

## Repo anchors (Autocodex)
- REPO_ROOT: `.`

## When to use
- Preparing commits for changes.

## Preconditions
- Git working tree exists.
- No destructive git commands.

## Inputs to confirm
- Intended change type (feat/fix/docs/refactor/test/chore)
- Scope (cli/plugins/api/ui/docs)

## Required artifacts
- Proposed conventional commit message
- Staged file list

## Quick path
- Inspect diff.
- Stage one intent.
- Commit with conventional message.

## Steps
1) `git status -sb`
2) Stage files for one intent.
3) `git commit -m "type(scope): subject"`

## Failure modes and responses
- **Mixed changes**: split into multiple commits.
- **Generated noise**: unstage or ignore.

## Definition of done
- Commit is conventional and scoped.

## Example (minimal)
- `feat(cli): add plugins command`
