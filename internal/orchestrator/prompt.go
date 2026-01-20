package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
)

func (o *Orchestrator) buildPrompt(phase, runID, feedback string) string {
	skillName := skillForPhase(phase)
	skillPath := ""
	if skillName != "" {
		skill, err := o.Skills.LoadSkill(skillName)
		if err == nil {
			skillPath = skill.Path
		}
	}

	var b strings.Builder
	b.WriteString("autocodex run ID: ")
	b.WriteString(runID)
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase)
	b.WriteString("\n")
	if skillName != "" {
		b.WriteString("Requested skill: ")
		b.WriteString(skillName)
		b.WriteString("\n\n")
	}
	if skillPath != "" {
		b.WriteString("Skill file: ")
		b.WriteString(skillPath)
		b.WriteString("\n")
		b.WriteString("Read the skill file and follow it before responding.\n")
	}
	if o.Config.Autonomy.Enabled {
		b.WriteString("\nAutonomy mode:\n")
		b.WriteString("- Do not ask follow-up questions.\n")
		b.WriteString("- If inputs are missing or ambiguous, make reasonable assumptions and proceed.\n")
		b.WriteString("- If you cannot proceed, state the blocking issue clearly in the output.\n")
		b.WriteString("- If skill instructions conflict with autonomy mode, autonomy mode takes precedence.\n")
		b.WriteString("- This is non-interactive: you must NOT ask questions or request clarification.\n")
		b.WriteString("- If you would ask a question, instead write a reasonable assumption and continue.\n")
	}
	if feedback != "" {
		b.WriteString("\n--- Feedback Context ---\n")
		b.WriteString(feedback)
		b.WriteString("\n--- End Feedback ---\n")
	}
	if o.Config.Autonomy.Enabled && strings.TrimSpace(o.Config.Autonomy.ActionsSchema) != "" {
		if schemaContent, err := os.ReadFile(filepath.Clean(o.Config.Autonomy.ActionsSchema)); err == nil {
			b.WriteString("\n--- Autonomy Actions Schema ---\n")
			b.WriteString(string(schemaContent))
			b.WriteString("\n--- End Autonomy Actions Schema ---\n")
		}
		b.WriteString("\nAutonomy actions:\n")
		b.WriteString("- Only output an ACTIONS JSON block in the test phase.\n")
		b.WriteString("- When the phase is test, append the following markers:\n")
		b.WriteString("ACTIONS_JSON_START\n")
		b.WriteString("<JSON that conforms to the schema>\n")
		b.WriteString("ACTIONS_JSON_END\n")
		b.WriteString("- If tests or acceptance criteria fail, set gates.blocking = true and include a stop.reason.\n")
	}
	return b.String()
}

func skillForPhase(phase string) string {
	switch phase {
	case "ideate":
		return "core-qna-synthesis"
	case "plan":
		return "core-holistic-planning-and-tracking"
	case "implement":
		return "eng-fullstack-engineer"
	case "review":
		return "eng-code-review-playbook"
	case "test":
		return "eng-smart-test-runner"
	default:
		return ""
	}
}
