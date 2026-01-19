---
name: core-ask-questions-if-underspecified
description: Clarify requirements when inputs are missing or ambiguous.
version: 0.2.0
---

# Ask Questions If Underspecified

## Repo anchors (autocodex)
- CLI_PATH: `cmd/autocodex/`
- INTERNAL_PATH: `internal/`
- PLUGINS_PATH: `plugins/`
- DOCS_PATH: `docs/`
- SKILLS_PATH: `skills/`
- UI_PATH: `web/`
- TEST_COMMANDS
  - Go: `go test ./...`
  - Go vet: `go vet ./...`
  - Go fmt: `gofmt -w $(rg --files -g '*.go')`

## When to use
- A request is ambiguous or missing required inputs (schemas, env vars, sample payloads, constraints).

## Preconditions
- You have the user’s request in the thread.

## Inputs to confirm
- Required schemas or config samples
- Runtime constraints (CLI vs API vs UI)
- Success criteria and scope boundaries

## Required artifacts
- A minimal list of questions
- A checklist of missing inputs

## Quick path
- Identify missing inputs.
- Ask the minimum number of questions to proceed.
- Provide a clear checklist.

## Steps
1) Stop before coding.
2) Ask 2–6 targeted questions.
3) Provide a short checklist of missing inputs.
4) State what you will do next once answered.

## Failure modes and responses
- **Too many questions**: reduce to only what blocks progress.
- **Vague questions**: rewrite as concrete inputs.

## Definition of done
- Missing inputs are explicitly listed and requested.

## Example (minimal)
- **Input**: “Add a plugin system.”
- **Questions**: What transport? What capability schema? Which paths?
- **Gotcha**: Don’t implement before protocol is defined.
