---
name: core-qna-synthesis
description: Refine multi-part questions and answer them with practical guidance.
version: 0.2.0
---

# Interpretive Q&A Synthesis

## Repo anchors (autocodex)
- CLI_PATH: `cmd/autocodex/`
- INTERNAL_PATH: `internal/`
- DOCS_PATH: `docs/`
- SKILLS_PATH: `skills/`

## When to use
- The user asks multiple related questions or a vague “big” question.

## Preconditions
- The questions are visible in the thread.
- If critical inputs are missing, STOP and use core-ask-questions-if-underspecified.

## Inputs to confirm
- Primary goal and success criteria
- Constraints (time, environment, tooling)

## Required artifacts
- Refined question list
- Practical answers with explicit assumptions
- Recommended next steps

## Quick path
- Restate and cluster questions.
- Add missing questions.
- Answer with assumptions and tradeoffs.
- Provide 3–7 concrete next steps.

## Steps
1) Restate and group questions.
2) Expand missing-but-required questions.
3) Answer each question clearly.
4) Provide recommendations and next steps.

## Failure modes and responses
- **Over-expansion**: keep added questions minimal.
- **Abstract answers**: replace with concrete actions.

## Definition of done
- Questions are refined and answered with practical guidance.

## Example (minimal)
- **Input**: “How should the plugin system work?”
- **Output**: Protocol choice, manifest rules, call flow, and next steps.
