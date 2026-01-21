package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type TasksFile struct {
	Version     string `json:"version"`
	GeneratedAt string `json:"generated_at"`
	SourcePlan  string `json:"source_plan"`
	Tasks       []Task `json:"tasks"`
}

type Task struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Goal               string     `json:"goal"`
	Scope              []string   `json:"scope"`
	Files              []string   `json:"files"`
	Dependencies       []string   `json:"dependencies"`
	Skills             []string   `json:"skills"`
	AcceptanceCriteria []string   `json:"acceptance_criteria"`
	Gates              *TaskGates `json:"gates,omitempty"`
	Tests              []string   `json:"tests"`
	Docs               []string   `json:"docs"`
	Rollout            string     `json:"rollout"`
	Observability      string     `json:"observability"`
	Status             string     `json:"status"`
	Priority           int        `json:"priority"`
	Owner              string     `json:"owner"`
	Notes              string     `json:"notes"`
}

type TaskGates struct {
	Tests        []string `json:"tests,omitempty"`
	Evidence     []string `json:"evidence,omitempty"`
	Verification []string `json:"verification,omitempty"`
}

const maxInvalidTasksExcerpt = 2000

func (c *Controller) generateTasksFile(ctx context.Context, task, slug, planPath, runTag string) (string, TasksFile, error) {
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

	tasksPath := tasksOutputPath(c.Config.Autonomy.TasksOutputTemplate, artifactSlug(slug, runTag))
	if err := os.MkdirAll(filepath.Dir(tasksPath), 0o755); err != nil {
		return "", TasksFile{}, fmt.Errorf("create tasks dir: %w", err)
	}

	beadPrefix := resolveBeadPrefix()
	prompt, err := c.buildTasksPrompt(task, string(planContent), string(schemaContent), planPath, tasksPath, beadPrefix)
	if err != nil {
		return "", TasksFile{}, err
	}

	usedFallback := false
	outputCtx, cancel := context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
	if c.Config.Codex.OutputLast {
		outputCtx = codex.WithOutputPath(outputCtx, tasksPath)
	}
	result, err := c.Codex.Exec(outputCtx, prompt)
	cancel()
	if err != nil {
		if strings.Contains(err.Error(), "requires follow-up") || strings.Contains(err.Error(), "needs_follow_up") {
			if c.Logger != nil {
				c.Logger.Warn("tasks generation requested follow-up, retrying without skill")
			}
			fallbackPrompt := buildFallbackTasksPrompt(task, string(planContent), string(schemaContent), planPath, tasksPath, beadPrefix)
			outputCtx, cancel = context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
			if c.Config.Codex.OutputLast {
				outputCtx = codex.WithOutputPath(outputCtx, tasksPath)
			}
			result, err = c.Codex.Exec(outputCtx, fallbackPrompt)
			cancel()
			usedFallback = true
		}
		if err != nil {
			if strings.Contains(err.Error(), "requires follow-up") {
				return c.writeFallbackTasks(task, planPath, tasksPath, beadPrefix)
			}
			return "", TasksFile{}, fmt.Errorf("tasks generation failed: %w", err)
		}
	}

	attempt := 1
	rawOutput := resolveOutput(tasksPath, result.Stdout)
	jsonPayload, err := extractJSON(rawOutput)
	if err != nil && !usedFallback {
		if c.Logger != nil {
			c.Logger.Warn("tasks JSON invalid, retrying without skill")
		}
		invalidPath := c.captureInvalidTasksOutput(rawOutput, tasksPath, attempt, err)
		fallbackPrompt := buildRetryTasksPrompt(task, string(planContent), string(schemaContent), planPath, tasksPath, beadPrefix, invalidPath, err, rawOutput)
		outputCtx, cancel = context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
		if c.Config.Codex.OutputLast {
			outputCtx = codex.WithOutputPath(outputCtx, tasksPath)
		}
		result, err = c.Codex.Exec(outputCtx, fallbackPrompt)
		cancel()
		if err != nil {
			if strings.Contains(err.Error(), "requires follow-up") {
				return c.writeFallbackTasks(task, planPath, tasksPath, beadPrefix)
			}
			return "", TasksFile{}, fmt.Errorf("tasks generation failed: %w", err)
		}
		usedFallback = true
		attempt++
		rawOutput = resolveOutput(tasksPath, result.Stdout)
		jsonPayload, err = extractJSON(rawOutput)
	}
	if err != nil {
		c.captureInvalidTasksOutput(rawOutput, tasksPath, attempt, err)
		if c.Config.Autonomy.AllowFallbackTasks != nil && !*c.Config.Autonomy.AllowFallbackTasks {
			return "", TasksFile{}, fmt.Errorf("tasks JSON invalid: %w", err)
		}
		return c.writeFallbackTasks(task, planPath, tasksPath, beadPrefix)
	}

	if err := validateJSONSchema(c.Config.Autonomy.TasksSchema, jsonPayload); err != nil {
		if c.Config.Autonomy.FailOnSchemaError != nil && !*c.Config.Autonomy.FailOnSchemaError {
			if c.Config.Autonomy.AllowFallbackTasks != nil && !*c.Config.Autonomy.AllowFallbackTasks {
				return "", TasksFile{}, fmt.Errorf("tasks schema validation failed: %w", err)
			}
			return c.writeFallbackTasks(task, planPath, tasksPath, beadPrefix)
		}
		return "", TasksFile{}, fmt.Errorf("tasks schema validation failed: %w", err)
	}

	var tasksFile TasksFile
	if err := json.Unmarshal([]byte(jsonPayload), &tasksFile); err != nil {
		return "", TasksFile{}, fmt.Errorf("parse tasks json: %w", err)
	}
	if len(tasksFile.Tasks) == 0 {
		return "", TasksFile{}, fmt.Errorf("no tasks returned by generator")
	}
	normalizeTasksFile(&tasksFile, beadPrefix)

	normalizedPayload, err := json.MarshalIndent(tasksFile, "", "  ")
	if err != nil {
		return "", TasksFile{}, fmt.Errorf("marshal tasks json: %w", err)
	}
	if err := os.WriteFile(tasksPath, append(normalizedPayload, '\n'), 0o644); err != nil {
		return "", TasksFile{}, fmt.Errorf("write tasks json: %w", err)
	}

	if c.Logger != nil {
		c.Logger.Info("tasks file generated", "tasks_path", tasksPath, "tasks", len(tasksFile.Tasks))
	}

	return tasksPath, tasksFile, nil
}

func normalizeTasksFile(tasksFile *TasksFile, prefix string) {
	if tasksFile == nil {
		return
	}
	prefix = sanitizeBeadPrefix(prefix)
	if prefix == "" {
		prefix = defaultBeadPrefix
	}
	idMap := map[string]string{}
	for i := range tasksFile.Tasks {
		task := &tasksFile.Tasks[i]
		if task.Scope == nil {
			task.Scope = []string{}
		}
		if task.Files == nil {
			task.Files = []string{}
		}
		if task.Dependencies == nil {
			task.Dependencies = []string{}
		}
		if task.Skills == nil {
			task.Skills = []string{}
		}
		if task.AcceptanceCriteria == nil {
			task.AcceptanceCriteria = []string{}
		}
		if task.Tests == nil {
			task.Tests = []string{}
		}
		if task.Docs == nil {
			task.Docs = []string{}
		}
		if task.Gates != nil {
			if task.Gates.Tests == nil {
				task.Gates.Tests = []string{}
			}
			if task.Gates.Evidence == nil {
				task.Gates.Evidence = []string{}
			}
			if task.Gates.Verification == nil {
				task.Gates.Verification = []string{}
			}
		}
		if strings.TrimSpace(task.Status) == "" {
			task.Status = "todo"
		}

		original := strings.TrimSpace(tasksFile.Tasks[i].ID)
		if original == "" {
			continue
		}
		normalized := normalizeTaskID(original, prefix)
		if normalized != original {
			idMap[original] = normalized
			tasksFile.Tasks[i].ID = normalized
		}
	}
	for i := range tasksFile.Tasks {
		deps := tasksFile.Tasks[i].Dependencies
		if len(deps) == 0 {
			continue
		}
		seen := map[string]bool{}
		normalizedDeps := make([]string, 0, len(deps))
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if mapped, ok := idMap[dep]; ok {
				dep = mapped
			}
			dep = normalizeTaskID(dep, prefix)
			if dep == "" || dep == tasksFile.Tasks[i].ID {
				continue
			}
			if seen[dep] {
				continue
			}
			seen[dep] = true
			normalizedDeps = append(normalizedDeps, dep)
		}
		tasksFile.Tasks[i].Dependencies = normalizedDeps
	}
}

func (c *Controller) buildTasksPrompt(task, planContent, schemaContent, planPath, tasksPath, beadPrefix string) (string, error) {
	skill, err := c.Skills.LoadSkill("core-holistic-planning-and-tracking")
	if err != nil {
		return "", fmt.Errorf("load skill core-holistic-planning-and-tracking: %w", err)
	}

	var b strings.Builder
	b.WriteString("autocodex autonomy task:\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n\n")
	b.WriteString("Requested skill: core-holistic-planning-and-tracking\n\n")
	b.WriteString("Skill file: ")
	b.WriteString(skill.Path)
	b.WriteString("\n")
	b.WriteString("Read the skill file and follow it before responding.\n\n")
	b.WriteString("Autonomy mode:\n")
	b.WriteString("- Do not ask follow-up questions.\n")
	b.WriteString("- If inputs are missing or ambiguous, make reasonable assumptions and proceed.\n")
	b.WriteString("- If you detect existing changes or pending change sets, assume they are intentional and proceed.\n")
	b.WriteString("- Ignore unrelated repo changes and focus only on the plan content provided.\n\n")
	b.WriteString("- If skill instructions conflict with autonomy mode, autonomy mode takes precedence.\n\n")
	b.WriteString("- This is non-interactive: you must NOT ask questions or request clarification.\n")
	b.WriteString("- If you would ask a question, instead write a reasonable assumption and continue.\n\n")
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
	b.WriteString("- Use bead IDs in the form ")
	b.WriteString(beadIDPattern(beadPrefix))
	b.WriteString(" for tasks (no extra dashes in <short>).\n")
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

func buildFallbackTasksPrompt(task, planContent, schemaContent, planPath, tasksPath, beadPrefix string) string {
	var b strings.Builder
	b.WriteString("autocodex autonomy task:\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n\n")
	b.WriteString("Autonomy mode:\n")
	b.WriteString("- Do not ask follow-up questions.\n")
	b.WriteString("- If inputs are missing or ambiguous, make reasonable assumptions and proceed.\n")
	b.WriteString("- Ignore unrelated repo changes and focus only on the plan content provided.\n")
	b.WriteString("- This is non-interactive: you must NOT ask questions or request clarification.\n")
	b.WriteString("- If you would ask a question, instead write a reasonable assumption and continue.\n\n")
	b.WriteString("Plan artifact:\n")
	b.WriteString("--- Plan Content ---\n")
	b.WriteString(planContent)
	b.WriteString("\n--- End Plan ---\n\n")
	b.WriteString("Tasks schema:\n")
	b.WriteString("--- Schema ---\n")
	b.WriteString(schemaContent)
	b.WriteString("\n--- End Schema ---\n\n")
	b.WriteString("Output requirements:\n")
	b.WriteString("- Produce JSON that validates against the schema.\n")
	b.WriteString("- Use version \"1.0\" and RFC3339 timestamps.\n")
	b.WriteString("- Use bead IDs in the form ")
	b.WriteString(beadIDPattern(beadPrefix))
	b.WriteString(" for tasks (no extra dashes in <short>).\n")
	b.WriteString("- Fill files, dependencies, acceptance_criteria, tests, docs when known.\n")
	b.WriteString("- Return JSON only (no code fences).\n")
	b.WriteString("- source_plan should be: ")
	b.WriteString(planPath)
	b.WriteString("\n")
	b.WriteString("- Target output path: ")
	b.WriteString(tasksPath)
	b.WriteString("\n")
	return b.String()
}

func buildRetryTasksPrompt(task, planContent, schemaContent, planPath, tasksPath, beadPrefix, invalidPath string, invalidErr error, invalidOutput string) string {
	base := buildFallbackTasksPrompt(task, planContent, schemaContent, planPath, tasksPath, beadPrefix)
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nPrevious output was invalid JSON.\n")
	if invalidErr != nil {
		b.WriteString("Error: ")
		b.WriteString(invalidErr.Error())
		b.WriteString("\n")
	}
	if invalidPath != "" {
		b.WriteString("Invalid output saved at: ")
		b.WriteString(invalidPath)
		b.WriteString("\n")
	}
	excerpt := truncateForPrompt(invalidOutput, maxInvalidTasksExcerpt)
	if excerpt != "" {
		b.WriteString("Invalid output excerpt:\n---\n")
		b.WriteString(excerpt)
		b.WriteString("\n---\n")
	}
	b.WriteString("Return corrected JSON only.\n")
	return b.String()
}

func (c *Controller) captureInvalidTasksOutput(raw, tasksPath string, attempt int, err error) string {
	if c.Config.Autonomy.KeepInvalidPayloads != nil && !*c.Config.Autonomy.KeepInvalidPayloads {
		return ""
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if attempt < 1 {
		attempt = 1
	}
	invalidPath := fmt.Sprintf("%s.invalid-%d.txt", tasksPath, attempt)
	if mkErr := os.MkdirAll(filepath.Dir(invalidPath), 0o755); mkErr != nil {
		if c.Logger != nil {
			c.Logger.Warn("failed to create invalid tasks output dir", "error", mkErr)
		}
		return ""
	}
	var b strings.Builder
	b.WriteString("generated_at: ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\n")
	if err != nil {
		b.WriteString("error: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	b.WriteString(trimmed)
	b.WriteString("\n")
	if writeErr := os.WriteFile(invalidPath, []byte(b.String()), 0o644); writeErr != nil {
		if c.Logger != nil {
			c.Logger.Warn("failed to write invalid tasks output", "error", writeErr)
		}
		return ""
	}
	if c.Logger != nil {
		c.Logger.Warn("invalid tasks output captured", "path", invalidPath)
	}
	return invalidPath
}

func truncateForPrompt(input string, maxChars int) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	if maxChars <= 0 || len(trimmed) <= maxChars {
		return trimmed
	}
	return trimmed[:maxChars] + "\n...[truncated]"
}

func (c *Controller) writeFallbackTasks(task, planPath, tasksPath, beadPrefix string) (string, TasksFile, error) {
	fallback := TasksFile{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SourcePlan:  planPath,
		Tasks: []Task{
			{
				ID:                 fallbackBeadID(beadPrefix),
				Title:              fmt.Sprintf("Execute task: %s", strings.TrimSpace(task)),
				Goal:               "Complete the requested task with available information.",
				Files:              []string{},
				AcceptanceCriteria: []string{"Autonomy loop completes without errors."},
				Status:             "todo",
			},
		},
	}
	payload, err := json.MarshalIndent(fallback, "", "  ")
	if err != nil {
		return "", TasksFile{}, fmt.Errorf("marshal fallback tasks: %w", err)
	}
	if err := os.WriteFile(tasksPath, append(payload, '\n'), 0o644); err != nil {
		return "", TasksFile{}, fmt.Errorf("write fallback tasks: %w", err)
	}
	if c.Logger != nil {
		c.Logger.Warn("fallback tasks written", "tasks_path", tasksPath)
	}
	return tasksPath, fallback, nil
}

func tasksOutputPath(template, slug string) string {
	if strings.Contains(template, "%s") {
		return fmt.Sprintf(template, slug)
	}
	return template
}

func extractJSON(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", fmt.Errorf("empty JSON output")
	}
	if fenced := extractJSONFromFences(trimmed); fenced != "" {
		return fenced, nil
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed, nil
	}
	if candidate := extractJSONFromText(trimmed); candidate != "" {
		return candidate, nil
	}
	return "", fmt.Errorf("invalid JSON returned by codex")
}

func extractJSONFromFences(output string) string {
	fences := strings.Split(output, "```")
	for i := 1; i < len(fences); i += 2 {
		block := strings.TrimSpace(fences[i])
		block = strings.TrimPrefix(block, "json")
		block = strings.TrimSpace(block)
		if json.Valid([]byte(block)) {
			return block
		}
	}
	return ""
}

func extractJSONFromText(output string) string {
	for start := 0; start < len(output); start++ {
		ch := output[start]
		if ch != '{' && ch != '[' {
			continue
		}
		for end := len(output); end > start; end-- {
			candidate := strings.TrimSpace(output[start:end])
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return ""
}

func validateJSONSchema(schemaPath, payload string) error {
	absPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("abs schema path: %w", err)
	}
	baseDir := filepath.Dir(absPath)
	tasksPath := filepath.Join(baseDir, "autonomy-tasks.schema.json")
	actionsPath := filepath.Join(baseDir, "autonomy-actions.schema.json")
	compiler := jsonschema.NewCompiler()
	compiler.LoadURL = func(url string) (io.ReadCloser, error) {
		switch url {
		case "https://autocodex.dev/contracts/autonomy-tasks.schema.json":
			return os.Open(tasksPath)
		case "https://autocodex.dev/contracts/autonomy-actions.schema.json":
			return os.Open(actionsPath)
		default:
			return jsonschema.LoadURL(url)
		}
	}
	schema, err := compiler.Compile("file://" + filepath.ToSlash(absPath))
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
