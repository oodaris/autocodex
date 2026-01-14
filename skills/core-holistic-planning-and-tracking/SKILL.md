---
name: core-holistic-planning-and-tracking
description: Create a plan and Beads tasks with dependencies and acceptance criteria.
version: 0.1.0
---

# Holistic Planning + Beads Tracking

Use when a plan and task breakdown are required.

## Preconditions
- `.beads/` exists.
- Missing inputs (schemas, env vars, sample payloads) must be requested first.

## Steps
1) Write a plan in `docs/plans/`.
2) Create Beads tasks using the template in `docs/AGENTS.md`.
3) Add dependencies via `bd dep add`.
4) Run `bd lint` if required.

## Output
- Plan file path
- Beads task list + dependencies
- Acceptance criteria for each task
