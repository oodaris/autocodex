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
	if c.Profile != "" && !oneOf(c.Profile, []string{"max_capability", "balanced", "max_throughput"}) {
		return fmt.Errorf("invalid profile: %s", c.Profile)
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
	if err := validateReasoningEffort(c.Codex.Model, c.Codex.ReasoningEffort); err != nil {
		return err
	}
	if strings.TrimSpace(c.Codex.Preset) != "" && strings.TrimSpace(c.Codex.CollaborationMode) == "" {
		return errors.New("codex.preset requires codex.collaboration_mode")
	}
	if c.Codex.CollaborationOn != nil && !*c.Codex.CollaborationOn {
		if strings.TrimSpace(c.Codex.CollaborationMode) != "" || strings.TrimSpace(c.Codex.Preset) != "" {
			return errors.New("codex.collaboration_enabled=false requires empty codex.collaboration_mode and codex.preset")
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
	if c.Loop.Feedback.MemoryMode != "" {
		if !oneOf(c.Loop.Feedback.MemoryMode, []string{"inline", "summary_ref", "ref_only"}) {
			return fmt.Errorf("invalid loop.feedback.memory_mode: %s", c.Loop.Feedback.MemoryMode)
		}
	}
	if c.Loop.Feedback.SnapshotMode != "" {
		if !oneOf(c.Loop.Feedback.SnapshotMode, []string{"inline", "summary_ref", "ref_only"}) {
			return fmt.Errorf("invalid loop.feedback.snapshot_mode: %s", c.Loop.Feedback.SnapshotMode)
		}
	}
	for _, source := range c.Loop.Feedback.Sources {
		if !oneOf(source, []string{"memory", "events", "artifacts", "snapshot"}) {
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
	if c.Loop.Feedback.SummaryMaxLines < 0 {
		return fmt.Errorf("invalid loop.feedback.summary_max_lines: %d", c.Loop.Feedback.SummaryMaxLines)
	}
	if c.Loop.PromptGuardrails.ReviewMaxBytes < 0 {
		return fmt.Errorf("invalid loop.prompt_guardrails.review_max_bytes: %d", c.Loop.PromptGuardrails.ReviewMaxBytes)
	}
	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("invalid api.port: %d", c.API.Port)
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		return errors.New("logging.level is required")
	}
	if !oneOf(strings.ToLower(c.Logging.Level), []string{"debug", "info", "warn", "error"}) {
		return fmt.Errorf("invalid logging.level: %s", c.Logging.Level)
	}
	if c.Logging.Format != "" && !oneOf(strings.ToLower(c.Logging.Format), []string{"json", "text"}) {
		return fmt.Errorf("invalid logging.format: %s", c.Logging.Format)
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
	if c.Autonomy.Coordinator.Strategy != "" {
		if !oneOf(c.Autonomy.Coordinator.Strategy, []string{"bead", "phase"}) {
			return fmt.Errorf("invalid autonomy.coordinator.strategy: %s", c.Autonomy.Coordinator.Strategy)
		}
	}
	if c.Autonomy.Coordinator.SelectionMode != "" {
		if !oneOf(c.Autonomy.Coordinator.SelectionMode, []string{"run_scoped", "all_ready"}) {
			return fmt.Errorf("invalid autonomy.coordinator.selection_mode: %s", c.Autonomy.Coordinator.SelectionMode)
		}
	}
	if c.Autonomy.Coordinator.MaxParallel != nil && *c.Autonomy.Coordinator.MaxParallel < 0 {
		return fmt.Errorf("invalid autonomy.coordinator.max_parallel: %d", *c.Autonomy.Coordinator.MaxParallel)
	}
	if c.Autonomy.Coordinator.FailureSummaryLimit < 0 {
		return fmt.Errorf("invalid autonomy.coordinator.failure_summary_limit: %d", c.Autonomy.Coordinator.FailureSummaryLimit)
	}
	if c.Autonomy.Gate.MaxFixAttemptsScope != "" {
		if !oneOf(c.Autonomy.Gate.MaxFixAttemptsScope, []string{"run", "global"}) {
			return fmt.Errorf("invalid autonomy.gate.max_fix_attempts_scope: %s", c.Autonomy.Gate.MaxFixAttemptsScope)
		}
	}
	if c.Autonomy.Harness.ImpactMode != "" {
		if !oneOf(c.Autonomy.Harness.ImpactMode, []string{"normal", "high"}) {
			return fmt.Errorf("invalid autonomy.harness.impact_mode: %s", c.Autonomy.Harness.ImpactMode)
		}
	}
	if c.Autonomy.Harness.StrictTrackingMode != "" {
		if !oneOf(c.Autonomy.Harness.StrictTrackingMode, []string{"bd_strict", "offline_evidence_only"}) {
			return fmt.Errorf("invalid autonomy.harness.strict_tracking_mode: %s", c.Autonomy.Harness.StrictTrackingMode)
		}
	}
	if c.Autonomy.Harness.Eval.MinScenarios < 0 {
		return fmt.Errorf("invalid autonomy.harness.eval.min_scenarios: %d", c.Autonomy.Harness.Eval.MinScenarios)
	}
	if c.Autonomy.Harness.Eval.MinPassRate < 0 || c.Autonomy.Harness.Eval.MinPassRate > 1 {
		return fmt.Errorf("invalid autonomy.harness.eval.min_pass_rate: %f", c.Autonomy.Harness.Eval.MinPassRate)
	}
	if c.Autonomy.Harness.Eval.MaxSoftFailures < 0 {
		return fmt.Errorf("invalid autonomy.harness.eval.max_soft_failures: %d", c.Autonomy.Harness.Eval.MaxSoftFailures)
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

func validateReasoningEffort(model, effort string) error {
	if effort == "" {
		return nil
	}
	allowed := []string{"minimal", "low", "medium", "high", "xhigh"}
	if !oneOf(effort, allowed) {
		return fmt.Errorf("invalid codex.reasoning_effort: %s", effort)
	}
	if model == "" {
		return nil
	}
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "gpt-5.1") {
		allowed = []string{"low", "medium", "high"}
	}
	if !oneOf(effort, allowed) {
		return fmt.Errorf("codex.reasoning_effort %q not supported for model %q (allowed: %s)", effort, model, strings.Join(allowed, ", "))
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
