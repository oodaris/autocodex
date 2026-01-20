package config

import (
	"errors"
	"fmt"
	"strings"
)

func (c Config) Validate() error {
	if c.Mode != "yolo" && c.Mode != "safe" {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}
	if c.Version != "v1" {
		return fmt.Errorf("unsupported config version: %s", c.Version)
	}
	if c.Codex.ApprovalPolicy != "" {
		if !oneOf(c.Codex.ApprovalPolicy, []string{"untrusted", "on-failure", "on-request", "never"}) {
			return fmt.Errorf("invalid codex.approval_policy: %s", c.Codex.ApprovalPolicy)
		}
	}
	if c.Codex.SandboxMode != "" {
		if !oneOf(c.Codex.SandboxMode, []string{"read-only", "workspace-write", "danger-full-access"}) {
			return fmt.Errorf("invalid codex.sandbox_mode: %s", c.Codex.SandboxMode)
		}
	}
	if c.Codex.ReasoningEffort != "" {
		if !oneOf(c.Codex.ReasoningEffort, []string{"minimal", "low", "medium", "high", "xhigh"}) {
			return fmt.Errorf("invalid codex.reasoning_effort: %s", c.Codex.ReasoningEffort)
		}
	}
	if c.Codex.JSONOutput && !c.Codex.OutputLast {
		return errors.New("codex.output_last_message must be true when codex.json_output is enabled")
	}
	if c.Cleanup.RetentionDays < 0 {
		return fmt.Errorf("invalid cleanup.retention_days: %d", c.Cleanup.RetentionDays)
	}
	if c.Hub.Enabled {
		seen := map[string]bool{}
		for _, ws := range c.Hub.Workspaces {
			if ws.ID == "" {
				return errors.New("hub.workspaces.id is required")
			}
			if strings.TrimSpace(ws.Root) == "" {
				return fmt.Errorf("hub workspace %s root is required", ws.ID)
			}
			if seen[ws.ID] {
				return fmt.Errorf("duplicate hub workspace id: %s", ws.ID)
			}
			seen[ws.ID] = true
		}
	}
	if c.Auth.Enabled && len(c.Auth.Tokens) == 0 && strings.TrimSpace(c.Auth.TokenEnv) == "" {
		return errors.New("auth.tokens or auth.token_env is required when auth is enabled")
	}
	if c.Loop.Mode != "" {
		if !oneOf(c.Loop.Mode, []string{"bounded", "continuous"}) {
			return fmt.Errorf("invalid loop.mode: %s", c.Loop.Mode)
		}
	}
	if c.Loop.StopConditions.MaxDurationSeconds < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_duration_seconds: %d", c.Loop.StopConditions.MaxDurationSeconds)
	}
	if c.Loop.StopConditions.MaxIdleSeconds < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_idle_seconds: %d", c.Loop.StopConditions.MaxIdleSeconds)
	}
	if c.Loop.StopConditions.MaxConsecutiveFailures < 0 {
		return fmt.Errorf("invalid loop.stop_conditions.max_consecutive_failures: %d", c.Loop.StopConditions.MaxConsecutiveFailures)
	}
	for phase, seconds := range c.Loop.PhaseIdleSecs {
		if seconds < 0 {
			return fmt.Errorf("invalid loop.phase_idle_seconds.%s: %d", phase, seconds)
		}
	}
	if c.Loop.Feedback.Mode != "" {
		if !oneOf(c.Loop.Feedback.Mode, []string{"off", "on"}) {
			return fmt.Errorf("invalid loop.feedback.mode: %s", c.Loop.Feedback.Mode)
		}
	}
	for _, source := range c.Loop.Feedback.Sources {
		if !oneOf(source, []string{"memory", "events", "artifacts"}) {
			return fmt.Errorf("invalid loop.feedback.sources entry: %s", source)
		}
	}
	if c.Loop.Feedback.MaxArtifacts < 0 {
		return fmt.Errorf("invalid loop.feedback.max_artifacts: %d", c.Loop.Feedback.MaxArtifacts)
	}
	if c.Loop.Feedback.MaxEvents < 0 {
		return fmt.Errorf("invalid loop.feedback.max_events: %d", c.Loop.Feedback.MaxEvents)
	}
	if c.Loop.Feedback.MaxBytes < 0 {
		return fmt.Errorf("invalid loop.feedback.max_bytes: %d", c.Loop.Feedback.MaxBytes)
	}
	if c.Loop.PromptGuardrails.ReviewMaxBytes < 0 {
		return fmt.Errorf("invalid loop.prompt_guardrails.review_max_bytes: %d", c.Loop.PromptGuardrails.ReviewMaxBytes)
	}
	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("invalid api.port: %d", c.API.Port)
	}
	if len(c.Loop.Phases) == 0 {
		return errors.New("loop.phases must not be empty")
	}
	if c.Autonomy.StopConditions.MaxFixAttempts < 0 {
		return fmt.Errorf("invalid autonomy.stop_conditions.max_fix_attempts: %d", c.Autonomy.StopConditions.MaxFixAttempts)
	}
	if c.Autonomy.StopConditions.MaxBeads < 0 {
		return fmt.Errorf("invalid autonomy.stop_conditions.max_beads: %d", c.Autonomy.StopConditions.MaxBeads)
	}
	if c.Autonomy.Enabled {
		if strings.TrimSpace(c.Autonomy.SpecTemplate) == "" {
			return errors.New("autonomy.spec_template is required when autonomy is enabled")
		}
		if strings.TrimSpace(c.Autonomy.PlanTemplate) == "" {
			return errors.New("autonomy.plan_template is required when autonomy is enabled")
		}
		if strings.TrimSpace(c.Autonomy.TasksSchema) == "" {
			return errors.New("autonomy.tasks_schema is required when autonomy is enabled")
		}
		if strings.TrimSpace(c.Autonomy.ActionsSchema) == "" {
			return errors.New("autonomy.actions_schema is required when autonomy is enabled")
		}
	}
	return nil
}

func oneOf(value string, allowed []string) bool {
	for _, v := range allowed {
		if value == v {
			return true
		}
	}
	return false
}
