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

	"github.com/oodaris/autocodex/internal/codex"
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
	if c.Config.Codex.OutputLast {
		specCtx = codex.WithOutputPath(specCtx, specOutputPath)
	}
	if c.Logger != nil {
		specCtx = codex.WithPIDReporter(specCtx, func(pid int) {
			c.Logger.Info("spec codex started", "pid", pid)
		})
	}
	specResult, err := c.Codex.Exec(specCtx, specPrompt)
	cancel()
	if err != nil {
		if strings.Contains(err.Error(), "requires follow-up") && strings.TrimSpace(specResult.Stdout) != "" {
			if c.Logger != nil {
				c.Logger.Warn("spec generation requested follow-up, using available output")
			}
			err = nil
		} else if strings.Contains(err.Error(), "needs_follow_up") {
			if c.Logger != nil {
				c.Logger.Warn("spec generation requested follow-up, retrying without skill")
			}
			fallbackPrompt := buildFallbackPrompt(task, string(specTemplate), specOutputPath)
			specCtx, cancel = context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
			if c.Config.Codex.OutputLast {
				specCtx = codex.WithOutputPath(specCtx, specOutputPath)
			}
			specResult, err = c.Codex.Exec(specCtx, fallbackPrompt)
			cancel()
		}
		if err != nil {
			return "", "", "", fmt.Errorf("spec generation failed: %w; stderr: %s", err, strings.TrimSpace(specResult.Stderr))
		}
	}
	planOutputCtx, cancelPlan := context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
	if c.Config.Codex.OutputLast {
		planOutputCtx = codex.WithOutputPath(planOutputCtx, planOutputPath)
	}
	if c.Logger != nil {
		planOutputCtx = codex.WithPIDReporter(planOutputCtx, func(pid int) {
			c.Logger.Info("plan codex started", "pid", pid)
		})
	}
	planResult, err := c.Codex.Exec(planOutputCtx, planPrompt)
	cancelPlan()
	if err != nil {
		if strings.Contains(err.Error(), "requires follow-up") && strings.TrimSpace(planResult.Stdout) != "" {
			if c.Logger != nil {
				c.Logger.Warn("plan generation requested follow-up, using available output")
			}
			err = nil
		} else if strings.Contains(err.Error(), "needs_follow_up") {
			if c.Logger != nil {
				c.Logger.Warn("plan generation requested follow-up, retrying without skill")
			}
			fallbackPrompt := buildFallbackPrompt(task, string(planTemplate), planOutputPath)
			planOutputCtx, cancelPlan = context.WithTimeout(ctx, time.Duration(c.Config.Codex.TimeoutSeconds)*time.Second)
			if c.Config.Codex.OutputLast {
				planOutputCtx = codex.WithOutputPath(planOutputCtx, planOutputPath)
			}
			planResult, err = c.Codex.Exec(planOutputCtx, fallbackPrompt)
			cancelPlan()
		}
		if err != nil {
			return "", "", "", fmt.Errorf("plan generation failed: %w; stderr: %s", err, strings.TrimSpace(planResult.Stderr))
		}
	}

	specOutput := resolveOutput(specOutputPath, specResult.Stdout)
	planOutput := resolveOutput(planOutputPath, planResult.Stdout)

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
	b.WriteString("Skill file: ")
	b.WriteString(skill.Path)
	b.WriteString("\n")
	b.WriteString("Read the skill file and follow it before responding.\n\n")
	b.WriteString("Autonomy mode:\n")
	b.WriteString("- Do not ask follow-up questions.\n")
	b.WriteString("- If inputs are missing or ambiguous, make reasonable assumptions and proceed.\n")
	b.WriteString("- If you encounter existing local changes, assume they are intentional and proceed.\n")
	b.WriteString("- Capture key assumptions in the output.\n\n")
	b.WriteString("- If skill instructions conflict with autonomy mode, autonomy mode takes precedence.\n\n")
	b.WriteString("- This is non-interactive: you must NOT ask questions or request clarification.\n")
	b.WriteString("- If you would ask a question, instead write a reasonable assumption and continue.\n\n")
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

func buildFallbackPrompt(task, template, outputPath string) string {
	var b strings.Builder
	b.WriteString("autocodex autonomy task:\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n\n")
	b.WriteString("Autonomy mode:\n")
	b.WriteString("- Do not ask follow-up questions.\n")
	b.WriteString("- If inputs are missing or ambiguous, make reasonable assumptions and proceed.\n")
	b.WriteString("- Capture key assumptions in the output.\n")
	b.WriteString("- This is non-interactive: you must NOT ask questions or request clarification.\n")
	b.WriteString("- If you would ask a question, instead write a reasonable assumption and continue.\n\n")
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
	return b.String()
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

func resolveOutput(path, fallback string) string {
	if strings.TrimSpace(path) != "" {
		if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) != "" {
			return string(data)
		}
	}
	return fallback
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
