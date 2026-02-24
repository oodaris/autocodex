package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestResolveBootstrapProfile(t *testing.T) {
	profile, explicit, err := resolveBootstrapProfile("", "")
	if err != nil {
		t.Fatalf("resolve default profile: %v", err)
	}
	if explicit {
		t.Fatalf("expected explicit to be false for default resolution")
	}
	if profile != profileMaxCapability {
		t.Fatalf("expected default profile %q, got %q", profileMaxCapability, profile)
	}

	profile, explicit, err = resolveBootstrapProfile("balanced", "")
	if err != nil {
		t.Fatalf("resolve explicit profile: %v", err)
	}
	if !explicit {
		t.Fatalf("expected explicit to be true")
	}
	if profile != profileBalanced {
		t.Fatalf("expected profile %q, got %q", profileBalanced, profile)
	}
}

func TestApplyBootstrapProfileMaxThroughput(t *testing.T) {
	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	if err := applyBootstrapProfile(&cfg, profileMaxThroughput); err != nil {
		t.Fatalf("apply profile: %v", err)
	}
	if cfg.Profile != profileMaxThroughput {
		t.Fatalf("expected profile %q, got %q", profileMaxThroughput, cfg.Profile)
	}
	if cfg.Codex.ReasoningEffort != "medium" {
		t.Fatalf("expected reasoning effort medium, got %q", cfg.Codex.ReasoningEffort)
	}
	if cfg.Codex.CollaborationOn == nil || *cfg.Codex.CollaborationOn {
		t.Fatalf("expected collaboration to be disabled")
	}
	if cfg.Codex.CollaborationMode != "" || cfg.Codex.Preset != "" {
		t.Fatalf("expected collaboration mode/preset cleared")
	}
	if cfg.Autonomy.Coordinator.MaxParallel == nil || *cfg.Autonomy.Coordinator.MaxParallel != 2 {
		t.Fatalf("expected max_parallel=2")
	}
	if cfg.Autonomy.StopConditions.MaxFixAttempts != 2 {
		t.Fatalf("expected max_fix_attempts=2")
	}
}

func TestBootstrapRepoWithProfileWritesConfig(t *testing.T) {
	base := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	if err := bootstrapRepo("autocodex.yaml", profileBalanced, false, false, false); err != nil {
		t.Fatalf("bootstrap repo: %v", err)
	}

	cfg, err := config.Load(filepath.Join(base, "autocodex.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Profile != profileBalanced {
		t.Fatalf("expected profile %q, got %q", profileBalanced, cfg.Profile)
	}
	if cfg.Codex.ReasoningEffort != "high" {
		t.Fatalf("expected reasoning effort high, got %q", cfg.Codex.ReasoningEffort)
	}
	if cfg.Codex.CollaborationOn == nil || !*cfg.Codex.CollaborationOn {
		t.Fatalf("expected collaboration enabled")
	}
	if cfg.Autonomy.Coordinator.MaxParallel == nil || *cfg.Autonomy.Coordinator.MaxParallel != 2 {
		t.Fatalf("expected coordinator max_parallel=2")
	}
}
