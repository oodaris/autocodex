package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	data := []byte("version: v1\nmode: yolo\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Codex.CLIPath == "" {
		t.Fatalf("expected codex cli path default")
	}
	if cfg.Profile != "max_capability" {
		t.Fatalf("expected profile default max_capability")
	}
	if cfg.Codex.ReasoningEffort == "" {
		t.Fatalf("expected codex reasoning effort default")
	}
	if cfg.Codex.CollaborationOn == nil || !*cfg.Codex.CollaborationOn {
		t.Fatalf("expected collaboration enabled default true")
	}
	if cfg.Codex.CollaborationMode == "" || cfg.Codex.Preset == "" {
		t.Fatalf("expected collaboration mode/preset defaults when enabled")
	}
	if cfg.Paths.StateDir == "" {
		t.Fatalf("expected state dir default")
	}
	if cfg.Loop.MaxIterations == 0 {
		t.Fatalf("expected loop max iterations default")
	}
	if cfg.Loop.Mode != "bounded" {
		t.Fatalf("expected loop mode default")
	}
	if cfg.Loop.Feedback.Mode != "off" {
		t.Fatalf("expected loop feedback mode default")
	}
	if len(cfg.Loop.Feedback.Sources) == 0 {
		t.Fatalf("expected loop feedback sources default")
	}
	if cfg.Autonomy.RequireActions == nil || *cfg.Autonomy.RequireActions {
		t.Fatalf("expected autonomy require_actions default false when autonomy disabled")
	}
	if cfg.Autonomy.RequireNext == nil || *cfg.Autonomy.RequireNext {
		t.Fatalf("expected autonomy require_next default false when autonomy disabled")
	}
	if cfg.Autonomy.RequireBD == nil || *cfg.Autonomy.RequireBD {
		t.Fatalf("expected autonomy require_bd default false when autonomy disabled")
	}
	if cfg.Autonomy.FailOnSchemaError == nil || !*cfg.Autonomy.FailOnSchemaError {
		t.Fatalf("expected autonomy fail_on_schema_error default true")
	}
	if cfg.Autonomy.AllowFallbackTasks == nil || !*cfg.Autonomy.AllowFallbackTasks {
		t.Fatalf("expected autonomy allow_fallback_tasks default true")
	}
	if cfg.Autonomy.KeepInvalidPayloads == nil || !*cfg.Autonomy.KeepInvalidPayloads {
		t.Fatalf("expected autonomy keep_invalid_payloads default true")
	}
	if cfg.Autonomy.Coordinator.Strategy == "" {
		t.Fatalf("expected autonomy coordinator strategy default")
	}
	if cfg.Autonomy.Coordinator.MaxParallel == nil || *cfg.Autonomy.Coordinator.MaxParallel == 0 {
		t.Fatalf("expected autonomy coordinator max_parallel default")
	}
	if cfg.Autonomy.Harness.ImpactMode != "normal" {
		t.Fatalf("expected harness impact_mode default normal")
	}
	if cfg.Autonomy.Harness.StrictTrackingMode != "bd_strict" {
		t.Fatalf("expected harness strict_tracking_mode default bd_strict")
	}
	if cfg.Autonomy.Harness.RequireCouncilOnHighImpact == nil || !*cfg.Autonomy.Harness.RequireCouncilOnHighImpact {
		t.Fatalf("expected harness require_council_on_high_impact default true")
	}
	if cfg.Autonomy.Harness.RequireIndependentCritic == nil || !*cfg.Autonomy.Harness.RequireIndependentCritic {
		t.Fatalf("expected harness require_independent_critic default true")
	}
	if cfg.Autonomy.Harness.RequireGateRunner == nil || !*cfg.Autonomy.Harness.RequireGateRunner {
		t.Fatalf("expected harness require_gate_runner default true")
	}
	if cfg.Autonomy.Harness.PreflightCommand == "" {
		t.Fatalf("expected harness preflight command default")
	}
	if cfg.Autonomy.Harness.RolePackPath != ".codex" {
		t.Fatalf("expected harness role pack path default .codex")
	}
	if cfg.Autonomy.Harness.Eval.Enabled == nil || !*cfg.Autonomy.Harness.Eval.Enabled {
		t.Fatalf("expected harness eval enabled default true")
	}
	if cfg.Autonomy.Harness.Eval.MinScenarios != 6 {
		t.Fatalf("expected harness eval min scenarios default 6")
	}
	if cfg.Autonomy.Harness.Eval.MinPassRate != 1.0 {
		t.Fatalf("expected harness eval min pass rate default 1.0")
	}
}

func TestValidateRejectsInvalidMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "bad"}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidProfile(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo", Profile: "ultra"}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidLoopMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Loop.Mode = "bad"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidLoggingLevel(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Logging.Level = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsPresetWithoutCollaborationMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Codex.Preset = "team"
	cfg.Codex.CollaborationMode = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestCollaborationDisabledClearsDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	data := []byte("version: v1\nmode: yolo\ncodex:\n  collaboration_enabled: false\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Codex.CollaborationOn == nil || *cfg.Codex.CollaborationOn {
		t.Fatalf("expected collaboration enabled false")
	}
	if cfg.Codex.CollaborationMode != "" || cfg.Codex.Preset != "" {
		t.Fatalf("expected collaboration mode/preset empty when disabled")
	}
}

func TestValidateRejectsCollaborationDisabledWithValues(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	disabled := false
	cfg.Codex.CollaborationOn = &disabled
	cfg.Codex.CollaborationMode = "auto"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidFeedbackSource(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Loop.Feedback.Sources = []string{"memory", "bad"}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidCoordinatorStrategy(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Autonomy.Coordinator.Strategy = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidHarnessImpactMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Autonomy.Harness.ImpactMode = "critical"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidHarnessTrackingMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Autonomy.Harness.StrictTrackingMode = "jira_strict"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsJSONWithoutOutputLast(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Codex.JSONOutput = true
	cfg.Codex.OutputLast = false
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsUnsupportedReasoningEffort(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Codex.ReasoningEffort = "banana"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsGpt51UnsupportedReasoningEffort(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Codex.Model = "gpt-5.1-codex"
	cfg.Codex.ReasoningEffort = "xhigh"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
	cfg.Codex.ReasoningEffort = "low"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestAutonomyDefaultsEnableFeedback(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	data := []byte("version: v1\nmode: yolo\nautonomy:\n  enabled: true\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Loop.Feedback.Mode != "on" {
		t.Fatalf("expected loop feedback mode on when autonomy enabled")
	}
	if cfg.Autonomy.RequireActions == nil || !*cfg.Autonomy.RequireActions {
		t.Fatalf("expected autonomy require_actions default true when autonomy enabled")
	}
	if cfg.Autonomy.RequireNext == nil || !*cfg.Autonomy.RequireNext {
		t.Fatalf("expected autonomy require_next default true when autonomy enabled")
	}
	if cfg.Autonomy.RequireBD == nil || !*cfg.Autonomy.RequireBD {
		t.Fatalf("expected autonomy require_bd default true when autonomy enabled")
	}
	if cfg.Autonomy.FailOnSchemaError == nil || !*cfg.Autonomy.FailOnSchemaError {
		t.Fatalf("expected autonomy fail_on_schema_error default true when autonomy enabled")
	}
	if cfg.Autonomy.AllowFallbackTasks == nil || !*cfg.Autonomy.AllowFallbackTasks {
		t.Fatalf("expected autonomy allow_fallback_tasks default true when autonomy enabled")
	}
	if cfg.Autonomy.KeepInvalidPayloads == nil || !*cfg.Autonomy.KeepInvalidPayloads {
		t.Fatalf("expected autonomy keep_invalid_payloads default true when autonomy enabled")
	}
}
