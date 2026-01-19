package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

const defaultSpecTemplate = "# <spec-title>\n" +
	"\n" +
	"## Metadata\n" +
	"```yaml\n" +
	"id: <slug>\n" +
	"owner:\n" +
	"status: draft\n" +
	"created: YYYY-MM-DD\n" +
	"updated: YYYY-MM-DD\n" +
	"```\n" +
	"\n" +
	"## Problem statement\n" +
	"Describe the user problem and why it matters.\n" +
	"\n" +
	"## Goals\n" +
	"- \n" +
	"\n" +
	"## Non-goals\n" +
	"- \n" +
	"\n" +
	"## Requirements\n" +
	"### Functional\n" +
	"- \n" +
	"\n" +
	"### Non-functional\n" +
	"- \n" +
	"\n" +
	"## Interfaces / data contracts\n" +
	"- \n" +
	"\n" +
	"## Acceptance criteria\n" +
	"- \n" +
	"\n" +
	"## Open questions\n" +
	"- \n" +
	"\n" +
	"## Risks\n" +
	"- \n" +
	"\n" +
	"## References\n" +
	"- \n"

const defaultPlanTemplate = "# <plan-title>\n" +
	"\n" +
	"## Metadata\n" +
	"```yaml\n" +
	"id: <slug>\n" +
	"spec: docs/specs/<slug>.md\n" +
	"owner:\n" +
	"status: draft\n" +
	"created: YYYY-MM-DD\n" +
	"updated: YYYY-MM-DD\n" +
	"```\n" +
	"\n" +
	"## Phases\n" +
	"- \n" +
	"\n" +
	"## Tasks (machine-readable)\n" +
	"- `docs/plans/<slug>-tasks.json` must conform to `docs/contracts/autonomy-tasks.schema.json`.\n" +
	"\n" +
	"## Task list (human summary)\n" +
	"| id | title | deps | status | notes |\n" +
	"| --- | --- | --- | --- | --- |\n" +
	"|  |  |  |  |  |\n" +
	"\n" +
	"## Risks\n" +
	"- \n" +
	"\n" +
	"## Evidence checklist\n" +
	"- \n" +
	"\n" +
	"## Rollout / rollback\n" +
	"- \n"

const defaultAutonomyTasksSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://autocodex.dev/contracts/autonomy-tasks.schema.json",
  "title": "autocodex autonomy tasks",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "generated_at", "source_plan", "tasks"],
  "properties": {
    "version": {
      "type": "string",
      "minLength": 1
    },
    "generated_at": {
      "type": "string",
      "format": "date-time"
    },
    "source_plan": {
      "type": "string",
      "minLength": 1
    },
    "tasks": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/task"
      }
    }
  },
  "definitions": {
    "task": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "title", "goal", "files", "acceptance_criteria"],
      "properties": {
        "id": {
          "type": "string",
          "minLength": 1
        },
        "title": {
          "type": "string",
          "minLength": 1
        },
        "goal": {
          "type": "string",
          "minLength": 1
        },
        "scope": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "files": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "dependencies": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "skills": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "acceptance_criteria": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "tests": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "docs": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "rollout": {
          "type": "string"
        },
        "observability": {
          "type": "string"
        },
        "status": {
          "type": "string",
          "enum": ["todo", "in_progress", "review", "done", "blocked"]
        },
        "priority": {
          "type": "integer",
          "minimum": 0
        },
        "owner": {
          "type": "string"
        },
        "notes": {
          "type": "string"
        }
      }
    }
  }
}
`

const defaultAutonomyActionsSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://autocodex.dev/contracts/autonomy-actions.schema.json",
  "title": "autocodex autonomy actions",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "summary", "next"],
  "properties": {
    "version": {
      "type": "string",
      "minLength": 1
    },
    "summary": {
      "type": "string",
      "minLength": 1
    },
    "next": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "enum": ["bead", "none"]
        },
        "id": {
          "type": "string",
          "minLength": 1
        },
        "reason": {
          "type": "string"
        }
      },
      "allOf": [
        {
          "if": {
            "properties": {
              "type": {"const": "bead"}
            }
          },
          "then": {
            "required": ["id"]
          }
        }
      ]
    },
    "updates": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "beads": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "required": ["id", "status"],
            "properties": {
              "id": {
                "type": "string",
                "minLength": 1
              },
              "status": {
                "type": "string",
                "enum": ["todo", "in_progress", "review", "done", "blocked"]
              },
              "note": {
                "type": "string"
              }
            }
          }
        }
      }
    },
    "create_beads": {
      "type": "array",
      "items": {
        "$ref": "autonomy-tasks.schema.json#/definitions/task"
      }
    },
    "gates": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "review_required": {
          "type": "boolean"
        },
        "tests": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "blocking": {
          "type": "boolean"
        }
      }
    },
    "stop": {
      "anyOf": [
        {"type": "null"},
        {
          "type": "object",
          "additionalProperties": false,
          "required": ["reason"],
          "properties": {
            "reason": {
              "type": "string",
              "minLength": 1
            },
            "details": {
              "type": "string"
            }
          }
        }
      ]
    }
  }
}
`

const defaultSkillAskQuestions = "---\n" +
	"name: core-ask-questions-if-underspecified\n" +
	"description: Clarify requirements when inputs are missing or ambiguous.\n" +
	"version: 0.1.0\n" +
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
	"version: 0.1.0\n" +
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
	"version: 0.1.0\n" +
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

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	force := fs.Bool("force", false, "overwrite existing templates, schemas, and skills")
	fs.Parse(args)

	if err := bootstrapRepo(*configPath, *force); err != nil {
		exitErr(err)
	}
	fmt.Printf("Bootstrap complete. Config: %s\n", *configPath)
}

func bootstrapRepo(configPath string, force bool) error {
	if err := ensureConfig(configPath); err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		return err
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		return err
	}

	skillsRoot := "skills"
	if err := bootstrapSkills(skillsRoot, force); err != nil {
		return err
	}
	if !skillsPathConfigured(cfg.Skills.Paths, skillsRoot) {
		fmt.Printf("Warning: skills.paths does not include %q; add it to use the bootstrap skill pack.\n", skillsRoot)
	}

	if cfg.Autonomy.Enabled {
		if err := bootstrapAutonomyAssets(cfg, force); err != nil {
			return err
		}
	}
	return nil
}

func bootstrapSkills(root string, force bool) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("skills root is empty")
	}
	skills := []struct {
		Name    string
		Content string
	}{
		{Name: "core-ask-questions-if-underspecified", Content: defaultSkillAskQuestions},
		{Name: "core-qna-synthesis", Content: defaultSkillQnaSynthesis},
		{Name: "core-holistic-planning-and-tracking", Content: defaultSkillHolisticPlanning},
	}
	for _, skill := range skills {
		path := filepath.Join(root, skill.Name, "SKILL.md")
		if err := writeFileIfMissing(path, []byte(skill.Content), force); err != nil {
			return err
		}
	}
	return nil
}

func skillsPathConfigured(paths []string, root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	root = filepath.Clean(root)
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if filepath.Clean(p) == root {
			return true
		}
	}
	return false
}

func bootstrapAutonomyAssets(cfg config.Config, force bool) error {
	if err := writeFileIfMissing(cfg.Autonomy.SpecTemplate, []byte(defaultSpecTemplate), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.PlanTemplate, []byte(defaultPlanTemplate), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.TasksSchema, []byte(defaultAutonomyTasksSchema), force); err != nil {
		return err
	}
	if err := writeFileIfMissing(cfg.Autonomy.ActionsSchema, []byte(defaultAutonomyActionsSchema), force); err != nil {
		return err
	}
	return nil
}

func writeFileIfMissing(path string, content []byte, force bool) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is empty")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
