package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func TestGatherFeedbackIncludesSources(t *testing.T) {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		t.Fatalf("ensure memory docs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.MemoryDir, "TODO.md"), []byte("remember this"), 0o644); err != nil {
		t.Fatalf("write memory doc: %v", err)
	}
	if err := store.AppendEvent(state.RunEvent{
		ID:      "evt-1",
		RunID:   run.ID,
		TS:      time.Now().UTC(),
		Type:    "phase_complete",
		Phase:   "ideate",
		Message: "done",
		Meta:    map[string]string{},
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "note.txt")
	if err := os.WriteFile(artifactPath, []byte("artifact content"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	var cfg config.Config
	cfg.ApplyDefaults()
	cfg.Loop.Feedback.Mode = "on"
	cfg.Loop.Feedback.Sources = []string{"memory", "events", "artifacts"}

	orch := Orchestrator{Config: cfg, Store: store}
	text, meta, err := orch.gatherFeedback(run.ID)
	if err != nil {
		t.Fatalf("gather feedback: %v", err)
	}
	if !strings.Contains(text, "Memory: TODO.md") {
		t.Fatalf("expected memory section")
	}
	if !strings.Contains(text, "Recent Events") {
		t.Fatalf("expected events section")
	}
	if !strings.Contains(text, "Recent Artifacts") {
		t.Fatalf("expected artifacts section")
	}
	if len(meta.MemoryDocs) == 0 || len(meta.EventIDs) == 0 || len(meta.ArtifactIDs) == 0 {
		t.Fatalf("expected feedback metadata")
	}
}

func TestShouldStopMaxIterations(t *testing.T) {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	var cfg config.Config
	cfg.ApplyDefaults()
	cfg.Loop.Mode = "continuous"
	cfg.Loop.MaxIterations = 2
	orch := Orchestrator{Config: cfg, Store: store}

	run.Iterations = 2
	stop, reason, status, _ := orch.shouldStop(context.Background(), run, time.Now().Add(-10*time.Second), time.Now(), 0)
	if !stop {
		t.Fatalf("expected stop")
	}
	if reason != "max_iterations" {
		t.Fatalf("unexpected reason: %s", reason)
	}
	if status != "completed" {
		t.Fatalf("unexpected status: %s", status)
	}
}

func TestShouldStopControlAction(t *testing.T) {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	action := "stop"
	if err := store.SaveRunControl(state.RunControl{
		RunID:      run.ID,
		Status:     "running",
		LastAction: &action,
	}); err != nil {
		t.Fatalf("save run control: %v", err)
	}

	var cfg config.Config
	cfg.ApplyDefaults()
	orch := Orchestrator{Config: cfg, Store: store}

	stop, reason, status, lastAction := orch.shouldStop(context.Background(), run, time.Now(), time.Now(), 0)
	if !stop {
		t.Fatalf("expected stop")
	}
	if reason != "stop" || status != "canceled" {
		t.Fatalf("unexpected stop decision: %s/%s", reason, status)
	}
	if lastAction == nil || *lastAction != "stop" {
		t.Fatalf("expected last action")
	}
}

func TestFinalizeRunAppendsProgress(t *testing.T) {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		t.Fatalf("ensure memory docs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	orch := Orchestrator{Store: store}
	orch.finalizeRun(run, "completed", nil, nil, nil)

	doc, err := store.GetMemoryDoc("PROGRESS.md")
	if err != nil {
		t.Fatalf("get memory doc: %v", err)
	}
	if !strings.Contains(doc.Content, run.ID) {
		t.Fatalf("expected run summary to include run id")
	}
}

func TestAppendPhaseSummary(t *testing.T) {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		t.Fatalf("ensure memory docs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	orch := Orchestrator{Store: store}
	started := time.Now().UTC()
	finished := started.Add(2 * time.Second)
	if err := orch.appendPhaseSummary(run.ID, "plan", started, finished, "hello"); err != nil {
		t.Fatalf("append phase summary: %v", err)
	}

	doc, err := store.GetMemoryDoc("PROGRESS.md")
	if err != nil {
		t.Fatalf("get memory doc: %v", err)
	}
	if !strings.Contains(doc.Content, "Phase plan") || !strings.Contains(doc.Content, "Output bytes") {
		t.Fatalf("expected phase summary content")
	}
}
