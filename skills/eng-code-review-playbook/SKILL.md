---
name: eng-code-review-playbook
description: Review changes for correctness, safety, and regressions.
version: 0.1.0
---

# Code Review Playbook

## Repo anchors (Autocodex)
- INTERNAL_PATH: `internal/`
- DOCS_PATH: `docs/`

## When to use
- Reviewing PRs or local diffs before merge.

## Preconditions
- You can see the diff and tests run.

## Inputs to confirm
- Intended behavior change
- Test evidence

## Required artifacts
- Findings list ordered by severity
- Open questions
- Short summary

## Quick path
- Scan for correctness, then safety, then tests.

## Steps
1) Identify behavior changes and edge cases.
2) Verify observability and error handling.
3) Check tests and docs.

## Failure modes and responses
- **No tests**: flag as blocking.
- **Hidden behavior changes**: request clarification.

## Definition of done
- Findings are clear, actionable, and prioritized.

## Example (minimal)
- **Finding**: Missing error handling in plugin handshake.
