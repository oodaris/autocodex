package orchestrator

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

type fakeExecutor struct {
	called int
}

func (f *fakeExecutor) Exec(ctx context.Context, prompt string) (codex.ExecResult, error) {
	f.called++
	return codex.ExecResult{Stdout: "ok"}, nil
}

type orchestratorLogRecord struct {
	msg   string
	attrs map[string]any
}

type orchestratorLogStore struct {
	mu      sync.Mutex
	records []orchestratorLogRecord
}

type orchestratorCaptureHandler struct {
	store *orchestratorLogStore
	base  []slog.Attr
}

func (h *orchestratorCaptureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *orchestratorCaptureHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{}
	for _, attr := range h.base {
		attrs[attr.Key] = attr.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.store.mu.Lock()
	h.store.records = append(h.store.records, orchestratorLogRecord{msg: r.Message, attrs: attrs})
	h.store.mu.Unlock()
	return nil
}

func (h *orchestratorCaptureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Attr, 0, len(h.base)+len(attrs))
	next = append(next, h.base...)
	next = append(next, attrs...)
	return &orchestratorCaptureHandler{store: h.store, base: next}
}

func (h *orchestratorCaptureHandler) WithGroup(name string) slog.Handler {
	_ = name
	return h
}

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

func TestRunLoggingIncludesRunIDAndStage(t *testing.T) {
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

	logStore := &orchestratorLogStore{}
	logger := slog.New(&orchestratorCaptureHandler{store: logStore})

	var cfg config.Config
	cfg.ApplyDefaults()
	cfg.Loop.Mode = "single"
	cfg.Loop.Phases = []string{"ideate"}
	cfg.Codex.TimeoutSeconds = 1

	exec := &fakeExecutor{}
	orch := Orchestrator{Config: cfg, Store: store, Logger: logger, Codex: exec}
	run, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var runStart, phaseStart *orchestratorLogRecord
	for i := range logStore.records {
		rec := logStore.records[i]
		switch rec.msg {
		case "run started":
			runStart = &logStore.records[i]
		case "phase start":
			phaseStart = &logStore.records[i]
		}
	}
	if runStart == nil || phaseStart == nil {
		t.Fatalf("expected run and phase log records")
	}
	if runStart.attrs["run_id"] != run.ID {
		t.Fatalf("expected run_id %s, got %v", run.ID, runStart.attrs["run_id"])
	}
	if runStart.attrs["stage"] != "run" {
		t.Fatalf("expected stage run, got %v", runStart.attrs["stage"])
	}
	if phaseStart.attrs["run_id"] != run.ID {
		t.Fatalf("expected phase run_id %s, got %v", run.ID, phaseStart.attrs["run_id"])
	}
	if phaseStart.attrs["stage"] != "ideate" {
		t.Fatalf("expected stage ideate, got %v", phaseStart.attrs["stage"])
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

func TestReviewGuardrailSkipsExec(t *testing.T) {
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

	var cfg config.Config
	cfg.ApplyDefaults()
	cfg.Loop.Phases = []string{"review"}
	cfg.Loop.PromptGuardrails.ReviewMaxBytes = 1
	cfg.Loop.Feedback.Mode = "off"

	exec := &fakeExecutor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	orch := Orchestrator{Config: cfg, Store: store, Logger: logger, Codex: exec}
	run, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if exec.called != 0 {
		t.Fatalf("expected exec to be skipped")
	}
	if run.Status != "completed" {
		t.Fatalf("expected run completed, got %s", run.Status)
	}

	skippedPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "review-skipped.txt")
	if _, err := os.Stat(skippedPath); err != nil {
		t.Fatalf("expected review-skipped.txt: %v", err)
	}
}
