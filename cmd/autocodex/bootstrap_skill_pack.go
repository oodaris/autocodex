package main

const defaultSkillAskQuestions = "---\n" +
	"name: core-ask-questions-if-underspecified\n" +
	"description: Clarify requirements when inputs are missing or ambiguous.\n" +
	"version: 0.2.0\n" +
	"---\n" +
	"\n" +
	"# Ask Questions If Underspecified\n" +
	"\n" +
	"## Repo anchors (autocodex)\n" +
	"- CLI_PATH: `cmd/autocodex/`\n" +
	"- INTERNAL_PATH: `internal/`\n" +
	"- PLUGINS_PATH: `plugins/`\n" +
	"- DOCS_PATH: `docs/`\n" +
	"- SKILLS_PATH: `skills/`\n" +
	"- UI_PATH: `web/`\n" +
	"- TEST_COMMANDS\n" +
	"  - Go: `go test ./...`\n" +
	"  - Go vet: `go vet ./...`\n" +
	"  - Go fmt: `gofmt -w $(rg --files -g '*.go')`\n" +
	"\n" +
	"## When to use\n" +
	"- A request is ambiguous or missing required inputs (schemas, env vars, sample payloads, constraints).\n" +
	"\n" +
	"## Preconditions\n" +
	"- You have the user’s request in the thread.\n" +
	"\n" +
	"## Inputs to confirm\n" +
	"- Required schemas or config samples\n" +
	"- Runtime constraints (CLI vs API vs UI)\n" +
	"- Success criteria and scope boundaries\n" +
	"\n" +
	"## Required artifacts\n" +
	"- A minimal list of questions\n" +
	"- A checklist of missing inputs\n" +
	"\n" +
	"## Quick path\n" +
	"- Identify missing inputs.\n" +
	"- Ask the minimum number of questions to proceed.\n" +
	"- Provide a clear checklist.\n" +
	"\n" +
	"## Steps\n" +
	"1) Stop before coding.\n" +
	"2) Ask 2–6 targeted questions.\n" +
	"3) Provide a short checklist of missing inputs.\n" +
	"4) State what you will do next once answered.\n" +
	"\n" +
	"## Failure modes and responses\n" +
	"- **Too many questions**: reduce to only what blocks progress.\n" +
	"- **Vague questions**: rewrite as concrete inputs.\n" +
	"\n" +
	"## Definition of done\n" +
	"- Missing inputs are explicitly listed and requested.\n" +
	"\n" +
	"## Example (minimal)\n" +
	"- **Input**: “Add a plugin system.”\n" +
	"- **Questions**: What transport? What capability schema? Which paths?\n" +
	"- **Gotcha**: Don’t implement before protocol is defined.\n"

const defaultSkillQnaSynthesis = "---\n" +
	"name: core-qna-synthesis\n" +
	"description: Refine multi-part questions and answer them with practical guidance.\n" +
	"version: 0.2.0\n" +
	"---\n" +
	"\n" +
	"# Interpretive Q&A Synthesis\n" +
	"\n" +
	"## Repo anchors (autocodex)\n" +
	"- CLI_PATH: `cmd/autocodex/`\n" +
	"- INTERNAL_PATH: `internal/`\n" +
	"- DOCS_PATH: `docs/`\n" +
	"- SKILLS_PATH: `skills/`\n" +
	"\n" +
	"## When to use\n" +
	"- The user asks multiple related questions or a vague “big” question.\n" +
	"\n" +
	"## Preconditions\n" +
	"- The questions are visible in the thread.\n" +
	"- If critical inputs are missing, STOP and use core-ask-questions-if-underspecified.\n" +
	"\n" +
	"## Inputs to confirm\n" +
	"- Primary goal and success criteria\n" +
	"- Constraints (time, environment, tooling)\n" +
	"\n" +
	"## Required artifacts\n" +
	"- Refined question list\n" +
	"- Practical answers with explicit assumptions\n" +
	"- Recommended next steps\n" +
	"\n" +
	"## Quick path\n" +
	"- Restate and cluster questions.\n" +
	"- Add missing questions.\n" +
	"- Answer with assumptions and tradeoffs.\n" +
	"- Provide 3–7 concrete next steps.\n" +
	"\n" +
	"## Steps\n" +
	"1) Restate and group questions.\n" +
	"2) Expand missing-but-required questions.\n" +
	"3) Answer each question clearly.\n" +
	"4) Provide recommendations and next steps.\n" +
	"\n" +
	"## Failure modes and responses\n" +
	"- **Over-expansion**: keep added questions minimal.\n" +
	"- **Abstract answers**: replace with concrete actions.\n" +
	"\n" +
	"## Definition of done\n" +
	"- Questions are refined and answered with practical guidance.\n" +
	"\n" +
	"## Example (minimal)\n" +
	"- **Input**: “How should the plugin system work?”\n" +
	"- **Output**: Protocol choice, manifest rules, call flow, and next steps.\n"

const defaultSkillHolisticPlanning = "---\n" +
	"name: core-holistic-planning-and-tracking\n" +
	"description: Create a plan and Beads tasks with dependencies and acceptance criteria.\n" +
	"version: 0.2.0\n" +
	"---\n" +
	"\n" +
	"# Holistic Planning + Beads Tracking\n" +
	"\n" +
	"## Repo anchors (autocodex)\n" +
	"- DOCS_PATH: `docs/`\n" +
	"- PLANS_PATH: `docs/plans/`\n" +
	"- BEADS_PATH: `.beads/`\n" +
	"\n" +
	"## When to use\n" +
	"- A plan and task breakdown are required before implementation.\n" +
	"\n" +
	"## Preconditions\n" +
	"- `.beads/` exists.\n" +
	"- If required inputs (schemas, env vars, sample payloads) are missing, STOP and ask.\n" +
	"\n" +
	"## Inputs to confirm\n" +
	"- Problem statement + success criteria\n" +
	"- Scope and constraints\n" +
	"- Required contracts (config, OpenAPI, protocols)\n" +
	"\n" +
	"## Required artifacts\n" +
	"- Plan file in `docs/plans/`\n" +
	"- Beads tasks with dependencies and acceptance criteria\n" +
	"\n" +
	"## Quick path\n" +
	"- Draft plan in `docs/plans/`.\n" +
	"- Create Beads tasks using the template in `docs/AGENTS.md`.\n" +
	"- Add dependencies with `bd dep add`.\n" +
	"\n" +
	"## Steps\n" +
	"1) Write or update the plan.\n" +
	"2) Create Beads tasks per work package.\n" +
	"3) Add dependencies.\n" +
	"4) Run `bd lint` if required.\n" +
	"\n" +
	"## Failure modes and responses\n" +
	"- **Missing inputs**: stop and request a checklist.\n" +
	"- **No Beads**: run `bd init` or ask the user to initialize.\n" +
	"\n" +
	"## Definition of done\n" +
	"- Plan exists and Beads tasks are created with dependencies.\n" +
	"\n" +
	"## Example (minimal)\n" +
	"- **Plan**: `docs/plans/autocodex-v1-plan.md`\n" +
	"- **Tasks**: Contracts → CLI → Plugins → API → UI.\n"
