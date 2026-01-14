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
}

func TestValidateRejectsInvalidMode(t *testing.T) {
	cfg := Config{Version: "v1", Mode: "bad"}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}
