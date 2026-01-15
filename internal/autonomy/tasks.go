package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type TasksFile struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
	SourcePlan  string `json:"source_plan"`
	Tasks       []Task `json:"tasks"`
}

type Task struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Goal               string   `json:"goal"`
	Scope              []string `json:"scope"`
	Files              []string `json:"files"`
	Dependencies       []string `json:"dependencies"`
	Skills             []string `json:"skills"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Tests              []string `json:"tests"`
	Docs               []string `json:"docs"`
	Rollout            string   `json:"rollout"`
	Observability      string   `json:"observability"`
	Status             string   `json:"status"`
	Priority           int      `json:"priority"`
	Owner              string   `json:"owner"`
	Notes              string   `json:"notes"`
}

func (c *Controller) generateTasksFile(ctx context.Context, task, slug, planPath string) (string, TasksFile, error) {
	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return "", TasksFile{}, fmt.Errorf("read plan: %w", err)
	}
	if strings.TrimSpace(c.Config.Autonomy.TasksSchema) == "" {
		return "", TasksFile{}, fmt.Errorf("autonomy.tasks_schema is empty")
	}
	schemaContent, err := os.ReadFile(filepath.Clean(c.Config.Autonomy.TasksSchema))
	if err != nil {
		return "", TasksFile{}, fmt.Errorf("read tasks schema: %w", err)
	}

	tasksPath := uniquePath(tasksOutputPath(c.Config.Autonomy.TasksOutputTemplate, slug))
	if err := os.MkdirAll(filepath.Dir(tasksPath), 0o755); err != nil {
		return "", TasksFile{}, fmt.Errorf("create tasks dir: %w", err)
	}

	prompt, err := c.buildTasksPrompt(task, string(planContent), string(schemaContent), planPath, tasksPath)
	if err != nil {
		return "", TasksFile{}, err
	}

	outputCtx, cancel := context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
	output, err := c.Codex.Exec(outputCtx, prompt)
	cancel()
	if err != nil {
		return "", TasksFile{}, fmt.Errorf("tasks generation failed: %w", err)
	}

	jsonPayload, err := extractJSON(output)
	if err != nil {
		return "", TasksFile{}, err
	}

	if err := validateJSONSchema(c.Config.Autonomy.TasksSchema, jsonPayload); err != nil {
		return "", TasksFile{}, fmt.Errorf("tasks schema validation failed: %w", err)
	}

	var tasksFile TasksFile
	if err := json.Unmarshal([]byte(jsonPayload), &tasksFile); err != nil {
		return "", TasksFile{}, fmt.Errorf("parse tasks json: %w", err)
	}
	if len(tasksFile.Tasks) == 0 {
		return "", TasksFile{}, fmt.Errorf("no tasks returned by generator")
	}

	if err := os.WriteFile(tasksPath, []byte(jsonPayload+"\n"), 0o644); err != nil {
		return "", TasksFile{}, fmt.Errorf("write tasks json: %w", err)
	}

	if c.Logger != nil {
		c.Logger.Info("tasks file generated", "tasks_path", tasksPath, "tasks", len(tasksFile.Tasks))
	}

	return tasksPath, tasksFile, nil
}

func (c *Controller) buildTasksPrompt(task, planContent, schemaContent, planPath, tasksPath string) (string, error) {
	skill, err := c.Skills.LoadSkill("core-holistic-planning-and-tracking")
	if err != nil {
		return "", fmt.Errorf("load skill core-holistic-planning-and-tracking: %w", err)
	}

	var b strings.Builder
	b.WriteString("autocodex autonomy task:\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n\n")
	b.WriteString("Requested skill: core-holistic-planning-and-tracking\n\n")
	b.WriteString("--- Skill Content ---\n")
	b.WriteString(skill.Content)
	b.WriteString("\n--- End Skill Content ---\n\n")
	b.WriteString("Plan artifact:\n")
	b.WriteString("--- Plan Content ---\n")
	b.WriteString(planContent)
	b.WriteString("\n--- End Plan Content ---\n\n")
	b.WriteString("Schema (MUST conform):\n")
	b.WriteString("--- Tasks Schema ---\n")
	b.WriteString(schemaContent)
	b.WriteString("\n--- End Tasks Schema ---\n\n")
	b.WriteString("Output requirements:\n")
	b.WriteString("- Produce JSON that validates against the schema.\n")
	b.WriteString("- Use version \"1.0\" and RFC3339 timestamps.\n")
	b.WriteString("- Use bead IDs in the form autocodex-<short> for tasks.\n")
	b.WriteString("- Fill files, dependencies, acceptance_criteria, tests, docs when known.\n")
	b.WriteString("- Return JSON only (no code fences).\n")
	b.WriteString("- source_plan should be: ")
	b.WriteString(planPath)
	b.WriteString("\n")
	b.WriteString("- Write to: ")
	b.WriteString(tasksPath)
	b.WriteString("\n")
	return b.String(), nil
}

func tasksOutputPath(template, slug string) string {
	if strings.Contains(template, "%s") {
		return fmt.Sprintf(template, slug)
	}
	return template
}

func extractJSON(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
			lines = lines[:len(lines)-1]
		}
		trimmed = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	if !json.Valid([]byte(trimmed)) {
		return "", fmt.Errorf("invalid JSON returned by codex")
	}
	return trimmed, nil
}

func validateJSONSchema(schemaPath, payload string) error {
	absPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("abs schema path: %w", err)
	}
	schema, err := jsonschema.Compile("file://" + filepath.ToSlash(absPath))
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	var data any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
