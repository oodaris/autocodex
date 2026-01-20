package main

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
