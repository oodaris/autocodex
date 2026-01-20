package autonomy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

func TestAutonomyStopsOnMaxBeads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stop-conditions test uses a shell-based bd stub")
	}

	tempDir := t.TempDir()
	specTemplate := filepath.Join(tempDir, "spec-template.md")
	planTemplate := filepath.Join(tempDir, "plan-template.md")
	if err := os.WriteFile(specTemplate, []byte("# Spec Template\n"), 0o644); err != nil {
		t.Fatalf("write spec template: %v", err)
	}
	if err := os.WriteFile(planTemplate, []byte("# Plan Template\n"), 0o644); err != nil {
		t.Fatalf("write plan template: %v", err)
	}

	bdBin := filepath.Join(tempDir, "bd")
	bdState := filepath.Join(tempDir, "bd-state")
	if err := writeBDFake(bdBin); err != nil {
		t.Fatalf("write fake bd: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, origPath))
	t.Setenv("BD_STATE", bdState)

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = filepath.Join(tempDir, "state")
	cfg.Skills.Paths = []string{filepath.Join(root, "skills")}
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.TasksSchema = filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	cfg.Autonomy.ActionsSchema = filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	cfg.Autonomy.TasksOutputTemplate = filepath.Join(tempDir, "%s-tasks.json")
	cfg.Autonomy.RequireActions = boolPtr(false)
	cfg.Autonomy.RequireNext = boolPtr(false)
	cfg.Autonomy.RequireBD = boolPtr(true)
	cfg.Autonomy.StopConditions.MaxBeads = 1
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	exec := strictnessExecutor{
		tasksJSON:  tasksJSON([]string{"autocodex-smoke-a", "autocodex-smoke-b"}),
		testOutput: "tests ok\n",
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Max beads test"}); err != nil {
		t.Fatalf("controller run failed: %v", err)
	}

	content, err := os.ReadFile(bdState)
	if err != nil {
		t.Fatalf("read bd state: %v", err)
	}
	stateText := string(content)
	if !strings.Contains(stateText, "autocodex-smokea|done") {
		t.Fatalf("expected first bead done, got: %s", stateText)
	}
	if !strings.Contains(stateText, "autocodex-smokeb|todo") {
		t.Fatalf("expected second bead todo when max_beads=1, got: %s", stateText)
	}
}

func TestAutonomyStopOnGateFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stop-conditions test uses a shell-based bd stub")
	}

	tempDir := t.TempDir()
	specTemplate := filepath.Join(tempDir, "spec-template.md")
	planTemplate := filepath.Join(tempDir, "plan-template.md")
	if err := os.WriteFile(specTemplate, []byte("# Spec Template\n"), 0o644); err != nil {
		t.Fatalf("write spec template: %v", err)
	}
	if err := os.WriteFile(planTemplate, []byte("# Plan Template\n"), 0o644); err != nil {
		t.Fatalf("write plan template: %v", err)
	}

	bdBin := filepath.Join(tempDir, "bd")
	bdState := filepath.Join(tempDir, "bd-state")
	if err := writeBDFake(bdBin); err != nil {
		t.Fatalf("write fake bd: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, origPath))
	t.Setenv("BD_STATE", bdState)

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = filepath.Join(tempDir, "state")
	cfg.Skills.Paths = []string{filepath.Join(root, "skills")}
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.TasksSchema = filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	cfg.Autonomy.ActionsSchema = filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	cfg.Autonomy.TasksOutputTemplate = filepath.Join(tempDir, "%s-tasks.json")
	cfg.Autonomy.RequireActions = boolPtr(true)
	cfg.Autonomy.RequireNext = boolPtr(false)
	cfg.Autonomy.RequireBD = boolPtr(true)
	cfg.Autonomy.StopConditions.StopOnGateFailure = boolPtr(true)
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	exec := strictnessExecutor{
		tasksJSON:  tasksJSON([]string{"autocodex-smoke"}),
		testOutput: "tests ok\n",
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Gate failure stop test"}); err == nil || !strings.Contains(err.Error(), "autonomy gate failed") {
		t.Fatalf("expected gate failure error, got: %v", err)
	}

	content, err := os.ReadFile(bdState)
	if err != nil {
		t.Fatalf("read bd state: %v", err)
	}
	if !strings.Contains(string(content), "autocodex-smoke|blocked") {
		t.Fatalf("expected bead blocked on gate failure, got: %s", string(content))
	}
}

func TestAutonomyMaxFixAttempts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stop-conditions test uses a shell-based bd stub")
	}

	tempDir := t.TempDir()
	specTemplate := filepath.Join(tempDir, "spec-template.md")
	planTemplate := filepath.Join(tempDir, "plan-template.md")
	if err := os.WriteFile(specTemplate, []byte("# Spec Template\n"), 0o644); err != nil {
		t.Fatalf("write spec template: %v", err)
	}
	if err := os.WriteFile(planTemplate, []byte("# Plan Template\n"), 0o644); err != nil {
		t.Fatalf("write plan template: %v", err)
	}

	bdBin := filepath.Join(tempDir, "bd")
	bdState := filepath.Join(tempDir, "bd-state")
	if err := writeBDFake(bdBin); err != nil {
		t.Fatalf("write fake bd: %v", err)
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, origPath))
	t.Setenv("BD_STATE", bdState)

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = filepath.Join(tempDir, "state")
	cfg.Skills.Paths = []string{filepath.Join(root, "skills")}
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.TasksSchema = filepath.Join(root, "docs/contracts/autonomy-tasks.schema.json")
	cfg.Autonomy.ActionsSchema = filepath.Join(root, "docs/contracts/autonomy-actions.schema.json")
	cfg.Autonomy.TasksOutputTemplate = filepath.Join(tempDir, "%s-tasks.json")
	cfg.Autonomy.RequireActions = boolPtr(true)
	cfg.Autonomy.RequireNext = boolPtr(false)
	cfg.Autonomy.RequireBD = boolPtr(true)
	cfg.Autonomy.StopConditions.MaxFixAttempts = 1
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	exec := strictnessExecutor{
		tasksJSON:  tasksJSON([]string{"autocodex-smoke"}),
		testOutput: "tests ok\n",
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Max fix attempts test"}); err == nil || !strings.Contains(err.Error(), "max fix attempts reached") {
		t.Fatalf("expected max fix attempts error, got: %v", err)
	}
}
