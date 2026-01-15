package autonomy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func (c *Controller) generateSpecAndPlan(ctx context.Context, task string) (string, string, string, error) {
	slug := slugify(task)
	if slug == "" {
		return "", "", "", errors.New("empty task slug")
	}

	specTemplatePath := filepath.Clean(c.Config.Autonomy.SpecTemplate)
	planTemplatePath := filepath.Clean(c.Config.Autonomy.PlanTemplate)
	specTemplate, err := os.ReadFile(specTemplatePath)
	if err != nil {
		return "", "", "", fmt.Errorf("read spec template: %w", err)
	}
	planTemplate, err := os.ReadFile(planTemplatePath)
	if err != nil {
		return "", "", "", fmt.Errorf("read plan template: %w", err)
	}

	specOutputPath := uniquePath(specOutputPath(specTemplatePath, slug))
	planOutputPath := uniquePath(planOutputPath(planTemplatePath, slug))

	if err := os.MkdirAll(filepath.Dir(specOutputPath), 0o755); err != nil {
		return "", "", "", fmt.Errorf("create spec dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(planOutputPath), 0o755); err != nil {
		return "", "", "", fmt.Errorf("create plan dir: %w", err)
	}

	specPrompt, err := c.buildPrompt(task, "core-qna-synthesis", string(specTemplate), specOutputPath)
	if err != nil {
		return "", "", "", err
	}
	planPrompt, err := c.buildPrompt(task, "core-holistic-planning-and-tracking", string(planTemplate), planOutputPath)
	if err != nil {
		return "", "", "", err
	}

	specCtx, cancel := context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
	specOutput, err := c.Codex.Exec(specCtx, specPrompt)
	cancel()
	if err != nil {
		return "", "", "", fmt.Errorf("spec generation failed: %w", err)
	}
	planOutputCtx, cancelPlan := context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
	planOutput, err := c.Codex.Exec(planOutputCtx, planPrompt)
	cancelPlan()
	if err != nil {
		return "", "", "", fmt.Errorf("plan generation failed: %w", err)
	}

	if err := os.WriteFile(specOutputPath, []byte(strings.TrimSpace(specOutput)+"\n"), 0o644); err != nil {
		return "", "", "", fmt.Errorf("write spec: %w", err)
	}
	if err := os.WriteFile(planOutputPath, []byte(strings.TrimSpace(planOutput)+"\n"), 0o644); err != nil {
		return "", "", "", fmt.Errorf("write plan: %w", err)
	}

	if c.Logger != nil {
		c.Logger.Info("autonomy artifacts written", "spec_path", specOutputPath, "plan_path", planOutputPath)
	}
	return specOutputPath, planOutputPath, slug, nil
}

func (c *Controller) buildPrompt(task, skillName, template, outputPath string) (string, error) {
	skill, err := c.Skills.LoadSkill(skillName)
	if err != nil {
		return "", fmt.Errorf("load skill %s: %w", skillName, err)
	}

	var b strings.Builder
	b.WriteString("autocodex autonomy task:\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n\n")
	b.WriteString("Requested skill: ")
	b.WriteString(skillName)
	b.WriteString("\n\n")
	b.WriteString("--- Skill Content ---\n")
	b.WriteString(skill.Content)
	b.WriteString("\n--- End Skill Content ---\n\n")
	b.WriteString("Template (fill in and return only the completed document):\n")
	b.WriteString("--- Template ---\n")
	b.WriteString(template)
	b.WriteString("\n--- End Template ---\n\n")
	b.WriteString("Output requirements:\n")
	b.WriteString("- Use Markdown.\n")
	b.WriteString("- Fill every section; use 'N/A' if a section is not applicable.\n")
	b.WriteString("- Return only the completed document; no commentary or code fences.\n")
	b.WriteString("- Target output path: ")
	b.WriteString(outputPath)
	b.WriteString("\n")
	return b.String(), nil
}

func latestTaskFromTodo(memoryDir string) (string, error) {
	if strings.TrimSpace(memoryDir) == "" {
		return "", errors.New("memory dir is empty")
	}
	path := filepath.Join(memoryDir, "TODO.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read TODO.md: %w", err)
	}
	content := string(data)
	idx := strings.LastIndex(content, "## Task")
	if idx == -1 {
		return "", errors.New("no task entries found in TODO.md")
	}
	segment := strings.TrimSpace(content[idx:])
	lines := strings.Split(segment, "\n")
	if len(lines) <= 1 {
		return "", errors.New("latest task entry is empty")
	}
	body := strings.TrimSpace(strings.Join(lines[1:], "\n"))
	return body, nil
}

func specOutputPath(specTemplate, slug string) string {
	dir := filepath.Dir(specTemplate)
	return filepath.Join(dir, slug+".md")
}

func planOutputPath(planTemplate, slug string) string {
	dir := filepath.Dir(planTemplate)
	return filepath.Join(dir, slug+"-plan.md")
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); err != nil {
			return candidate
		}
	}
}

func slugify(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	var b strings.Builder
	b.Grow(len(value))
	dash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		default:
			if !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "task"
	}
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
	}
	if slug == "" {
		return "task"
	}
	return slug
}
