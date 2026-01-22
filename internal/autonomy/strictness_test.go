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

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/logging"
	"github.com/oodaris/autocodex/internal/skills"
	"github.com/oodaris/autocodex/internal/state"
)

type strictnessExecutor struct {
	tasksJSON  string
	testOutput string
}

func (exec strictnessExecutor) Exec(_ context.Context, prompt string) (codex.ExecResult, error) {
	result := codex.ExecResult{}
	switch {
	case strings.Contains(prompt, "Schema (MUST conform)") && strings.Contains(prompt, "Tasks Schema"):
		result.Stdout = exec.tasksJSON
		return result, nil
	case strings.Contains(prompt, "Template (fill in"):
		result.Stdout = "# Spec\n\nok\n"
		return result, nil
	case strings.Contains(prompt, "Phase: test"):
		result.Stdout = exec.testOutput
		return result, nil
	default:
		result.Stdout = "ok\n"
		return result, nil
	}
}

func TestAutonomyRequiresActions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("strictness test uses a shell-based bd stub")
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
		Logger: logging.NewLogger("error", "json"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Require actions test"}); err == nil || !strings.Contains(err.Error(), "missing required ACTIONS output") {
		t.Fatalf("expected missing ACTIONS error, got: %v", err)
	}
	content, err := os.ReadFile(bdState)
	if err != nil {
		t.Fatalf("read bd state: %v", err)
	}
	if !strings.Contains(string(content), "autocodex-smoke|blocked") {
		t.Fatalf("expected bead blocked when actions missing, got: %s", string(content))
	}
}

func TestAutonomyRequiresNextWhenMultipleBeadsReady(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("strictness test uses a shell-based bd stub")
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
	cfg.Autonomy.RequireNext = boolPtr(true)
	cfg.Autonomy.RequireBD = boolPtr(true)
	cfg.Codex.TimeoutSeconds = 1
	cfg.Beads.Enabled = true
	cfg.Beads.AutoCreate = true
	cfg.Beads.AutoUpdate = true

	exec := strictnessExecutor{
		tasksJSON:  tasksJSON([]string{"autocodex-smoke-a", "autocodex-smoke-b", "autocodex-smoke-c"}),
		testOutput: "tests ok\nACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"done\",\"next\":{\"type\":\"none\"},\"updates\":{\"beads\":[{\"id\":\"autocodex-smoke-a\",\"status\":\"done\"}]},\"gates\":{\"review_required\":false,\"tests\":[],\"blocking\":false},\"stop\":null}\nACTIONS_JSON_END\n",
	}

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	loader := skills.Loader{Paths: cfg.Skills.Paths}
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error", "json"),
		Store:  store,
		Skills: loader,
		Codex:  exec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Require next test"}); err == nil || !strings.Contains(err.Error(), "explicit next bead required") {
		t.Fatalf("expected require-next error, got: %v", err)
	}
}

func TestAutonomyRequireBDFailsFast(t *testing.T) {
	tempDir := t.TempDir()
	specTemplate := filepath.Join(tempDir, "spec-template.md")
	planTemplate := filepath.Join(tempDir, "plan-template.md")
	if err := os.WriteFile(specTemplate, []byte("# Spec Template\n"), 0o644); err != nil {
		t.Fatalf("write spec template: %v", err)
	}
	if err := os.WriteFile(planTemplate, []byte("# Plan Template\n"), 0o644); err != nil {
		t.Fatalf("write plan template: %v", err)
	}

	t.Setenv("PATH", tempDir)

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Paths.StateDir = filepath.Join(tempDir, "state")
	cfg.Autonomy.Enabled = true
	cfg.Autonomy.SpecTemplate = specTemplate
	cfg.Autonomy.PlanTemplate = planTemplate
	cfg.Autonomy.RequireBD = boolPtr(true)
	cfg.Autonomy.RequireActions = boolPtr(false)
	cfg.Autonomy.RequireNext = boolPtr(false)

	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	controller := Controller{
		Config: cfg,
		Logger: logging.NewLogger("error", "json"),
		Store:  store,
		Skills: skills.Loader{},
		Codex:  strictnessExecutor{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := controller.Run(ctx, Input{Task: "Require bd test"}); err == nil || !strings.Contains(err.Error(), "bd is required for autonomy runs") {
		t.Fatalf("expected bd required error, got: %v", err)
	}
}

func tasksJSON(ids []string) string {
	var b strings.Builder
	b.WriteString(`{"version":"1.0","generated_at":"2026-01-15T00:00:00Z","source_plan":"plan.md","tasks":[`)
	for i, id := range ids {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":"%s","title":"Smoke bead","goal":"Verify autonomy loop","files":[],"acceptance_criteria":[]}`, id)
	}
	b.WriteString("]}")
	return b.String()
}

func boolPtr(value bool) *bool {
	return &value
}
