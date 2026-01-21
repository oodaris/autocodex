package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestResolveInputInline(t *testing.T) {
	got, err := resolveInput(`{"ok":true}`, "")
	if err != nil {
		t.Fatalf("resolve input: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("unexpected inline input: %s", string(got))
	}
}

func TestResolveInputFile(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "input.json")
	if err := os.WriteFile(path, []byte(`{"file":true}`), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}
	got, err := resolveInput("", path)
	if err != nil {
		t.Fatalf("resolve input file: %v", err)
	}
	if string(got) != `{"file":true}` {
		t.Fatalf("unexpected file input: %s", string(got))
	}
}

func TestResolveInputEmpty(t *testing.T) {
	got, err := resolveInput("", "")
	if err != nil {
		t.Fatalf("resolve input empty: %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("expected empty object, got %s", string(got))
	}
}

func TestAppendTaskToTodo(t *testing.T) {
	base := t.TempDir()
	memoryDir := filepath.Join(base, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("mkdir memory dir: %v", err)
	}
	todoPath := filepath.Join(memoryDir, "TODO.md")
	if err := os.WriteFile(todoPath, []byte("# TODO\n"), 0o644); err != nil {
		t.Fatalf("write TODO: %v", err)
	}
	if err := appendTaskToTodo(memoryDir, "ship it"); err != nil {
		t.Fatalf("append task: %v", err)
	}
	data, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatalf("read TODO: %v", err)
	}
	if !strings.Contains(string(data), "ship it") {
		t.Fatalf("expected task payload to be appended")
	}
}

func TestAppendTaskToTodoEmptyDir(t *testing.T) {
	if err := appendTaskToTodo("", "task"); err == nil {
		t.Fatalf("expected error for empty memory dir")
	}
}

func TestApplyTaskInput(t *testing.T) {
	base := t.TempDir()
	cfg := &config.Config{
		Paths: config.PathsConfig{
			StateDir:     base,
			MemoryDir:    "memory",
			LogsDir:      "logs",
			RunsDir:      "runs",
			ArtifactsDir: "artifacts",
		},
	}
	payload := "do the thing"
	out := captureStdout(t, func() {
		got, err := applyTaskInput(cfg, payload)
		if err != nil {
			t.Fatalf("apply task input: %v", err)
		}
		if got != payload {
			t.Fatalf("expected payload to be returned")
		}
	})
	if !strings.Contains(out, "Task appended") {
		t.Fatalf("expected task appended output")
	}
	if cfg.Loop.Feedback.Mode != "on" {
		t.Fatalf("expected feedback mode to be on")
	}
	todoPath := filepath.Join(cfg.MemoryDir(), "TODO.md")
	data, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatalf("read TODO: %v", err)
	}
	if !strings.Contains(string(data), payload) {
		t.Fatalf("expected payload in TODO.md")
	}
}

func TestApplyStartPhase(t *testing.T) {
	cfg := &config.Config{Loop: config.LoopConfig{Phases: []string{"ideate", "plan", "implement", "review", "test"}}}
	if err := applyStartPhase(cfg, "implement"); err != nil {
		t.Fatalf("applyStartPhase: %v", err)
	}
	if got := strings.Join(cfg.Loop.Phases, ","); got != "implement,review,test" {
		t.Fatalf("unexpected phases: %s", got)
	}
}

func TestApplyStartPhaseInvalid(t *testing.T) {
	cfg := &config.Config{Loop: config.LoopConfig{Phases: []string{"ideate", "plan"}}}
	if err := applyStartPhase(cfg, "test"); err == nil {
		t.Fatalf("expected error for invalid phase")
	}
}

func TestAppendArtifactHints(t *testing.T) {
	base := t.TempDir()
	specDir := filepath.Join(base, "specs")
	planDir := filepath.Join(base, "plans")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	specPath := filepath.Join(specDir, "spec.md")
	planPath := filepath.Join(planDir, "plan.md")
	if err := os.WriteFile(specPath, []byte("spec"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(planPath, []byte("plan"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	cfg := config.Config{
		Autonomy: config.AutonomyConfig{
			SpecTemplate: filepath.Join(specDir, "TEMPLATE.md"),
			PlanTemplate: filepath.Join(planDir, "TEMPLATE.md"),
		},
	}
	payload, err := appendArtifactHints("do work", cfg, "implement", true)
	if err != nil {
		t.Fatalf("appendArtifactHints: %v", err)
	}
	if !strings.Contains(payload, specPath) || !strings.Contains(payload, planPath) {
		t.Fatalf("expected spec/plan paths appended")
	}
}
