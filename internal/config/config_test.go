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
	if cfg.Codex.ReasoningEffort == "" {
		t.Fatalf("expected codex reasoning effort default")
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
}

func TestValidateRejectsInvalidMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "bad"}
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

func TestValidateRejectsInvalidFeedbackSource(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Loop.Feedback.Sources = []string{"memory", "bad"}
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
