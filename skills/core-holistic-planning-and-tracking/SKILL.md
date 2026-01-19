---
name: core-holistic-planning-and-tracking
description: Create a plan and Beads tasks with dependencies and acceptance criteria.
version: 0.2.0
---

# Holistic Planning + Beads Tracking

## Repo anchors (autocodex)
- DOCS_PATH: `docs/`
- PLANS_PATH: `docs/plans/`
- BEADS_PATH: `.beads/`

## When to use
- A plan and task breakdown are required before implementation.

## Preconditions
- `.beads/` exists.
- If required inputs (schemas, env vars, sample payloads) are missing, STOP and ask.

## Inputs to confirm
- Problem statement + success criteria
- Scope and constraints
- Required contracts (config, OpenAPI, protocols)

## Required artifacts
- Plan file in `docs/plans/`
- Beads tasks with dependencies and acceptance criteria

## Quick path
- Draft plan in `docs/plans/`.
- Create Beads tasks using the template in `docs/AGENTS.md`.
- Add dependencies with `bd dep add`.

## Steps
1) Write or update the plan.
2) Create Beads tasks per work package.
3) Add dependencies.
4) Run `bd lint` if required.

## Failure modes and responses
- **Missing inputs**: stop and request a checklist.
- **No Beads**: run `bd init` or ask the user to initialize.

## Definition of done
- Plan exists and Beads tasks are created with dependencies.

## Example (minimal)
- **Plan**: `docs/plans/autocodex-v1-plan.md`
- **Tasks**: Contracts → CLI → Plugins → API → UI.
