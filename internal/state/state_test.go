package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreInitAndRunLifecycle(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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
		t.Fatalf("memory docs: %v", err)
	}

	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if run.ID == "" {
		t.Fatalf("expected run id")
	}

	if _, err := os.Stat(filepath.Join(store.RunsDir, run.ID, "run.json")); err != nil {
		t.Fatalf("run.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.LogsDir, run.ID+".jsonl")); err != nil {
		t.Fatalf("events log missing: %v", err)
	}
}

func TestRunControlAndFeedbackPersistence(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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

	stopReason := "manual stop"
	lastError := "boom"
	control := RunControl{
		RunID:      run.ID,
		Status:     "running",
		StopReason: &stopReason,
		LastError:  &lastError,
	}
	if err := store.SaveRunControl(control); err != nil {
		t.Fatalf("save run control: %v", err)
	}
	loadedControl, err := store.GetRunControl(run.ID)
	if err != nil {
		t.Fatalf("get run control: %v", err)
	}
	if loadedControl == nil || loadedControl.RunID != run.ID {
		t.Fatalf("expected run control")
	}
	if loadedControl.StopReason == nil || *loadedControl.StopReason != stopReason {
		t.Fatalf("expected stop reason")
	}

	feedback := RunFeedback{
		RunID:             run.ID,
		Sources:           []string{"memory"},
		LastPromptSummary: "prompt summary",
		LastOutputSummary: "output summary",
		MemoryDocs:        []string{"TODO.md"},
	}
	if err := store.SaveRunFeedback(feedback); err != nil {
		t.Fatalf("save run feedback: %v", err)
	}
	loadedFeedback, err := store.GetRunFeedback(run.ID)
	if err != nil {
		t.Fatalf("get run feedback: %v", err)
	}
	if loadedFeedback == nil || loadedFeedback.RunID != run.ID {
		t.Fatalf("expected run feedback")
	}
	if len(loadedFeedback.MemoryDocs) != 1 {
		t.Fatalf("expected memory docs")
	}
}

func TestListRunsHydratesControl(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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

	reason := "shutdown"
	action := "stop"
	now := time.Now().UTC()
	control := RunControl{
		RunID:        run.ID,
		Status:       "failed",
		StopReason:   &reason,
		LastAction:   &action,
		LastActionAt: &now,
		UpdatedAt:    now,
	}
	if err := store.SaveRunControl(control); err != nil {
		t.Fatalf("save run control: %v", err)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	got := runs[0]
	if got.StopReason == nil || *got.StopReason != reason {
		t.Fatalf("expected stop reason")
	}
	if got.LastAction == nil || *got.LastAction != action {
		t.Fatalf("expected last action")
	}
	if got.Status != "failed" {
		t.Fatalf("expected status failed, got %s", got.Status)
	}

	loaded, err := store.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if loaded.StopReason == nil || *loaded.StopReason != reason {
		t.Fatalf("expected stop reason in run detail")
	}
}

func TestAppendMemoryDoc(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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

	if err := store.AppendMemoryDoc("PROGRESS.md", "Run summary line"); err != nil {
		t.Fatalf("append memory doc: %v", err)
	}
	doc, err := store.GetMemoryDoc("PROGRESS.md")
	if err != nil {
		t.Fatalf("get memory doc: %v", err)
	}
	if doc == nil || !strings.Contains(doc.Content, "Run summary line") {
		t.Fatalf("expected appended content")
	}
}

func TestCleanupRuns(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}

	oldRun, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	oldFinished := time.Now().UTC().Add(-48 * time.Hour)
	oldRun.FinishedAt = &oldFinished
	oldRun.Status = "completed"
	if err := store.SaveRun(oldRun); err != nil {
		t.Fatalf("save run: %v", err)
	}

	recentRun, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	recentFinished := time.Now().UTC().Add(-2 * time.Hour)
	recentRun.FinishedAt = &recentFinished
	recentRun.Status = "completed"
	if err := store.SaveRun(recentRun); err != nil {
		t.Fatalf("save run: %v", err)
	}

	oldLog := filepath.Join(store.LogsDir, oldRun.ID+".jsonl")
	if err := os.WriteFile(oldLog, []byte("log"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	recentLog := filepath.Join(store.LogsDir, recentRun.ID+".jsonl")
	if err := os.WriteFile(recentLog, []byte("log"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	opts := CleanupOptions{
		OlderThan: 24 * time.Hour,
		Now:       time.Now().UTC(),
	}
	result, err := store.CleanupRuns(opts)
	if err != nil {
		t.Fatalf("cleanup runs: %v", err)
	}
	if len(result.Deleted) != 1 {
		t.Fatalf("expected 1 run deleted, got %d", len(result.Deleted))
	}

	if _, err := os.Stat(filepath.Join(store.RunsDir, oldRun.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old run dir removed, err=%v", err)
	}
	if _, err := os.Stat(oldLog); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old log removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(store.RunsDir, recentRun.ID)); err != nil {
		t.Fatalf("expected recent run dir kept, err=%v", err)
	}
	if _, err := os.Stat(recentLog); err != nil {
		t.Fatalf("expected recent log kept, err=%v", err)
	}
}

func TestRunHeartbeat(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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
	if err := store.TouchRunHeartbeat(run.ID, 1234); err != nil {
		t.Fatalf("touch heartbeat: %v", err)
	}
	hb, err := store.GetRunHeartbeat(run.ID)
	if err != nil {
		t.Fatalf("get heartbeat: %v", err)
	}
	if hb == nil || hb.RunID != run.ID || hb.PID != 1234 {
		t.Fatalf("unexpected heartbeat")
	}
}

func TestRunLockLifecycle(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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

	lock, err := store.AcquireRunLock(run.ID)
	if err != nil {
		t.Fatalf("acquire run lock: %v", err)
	}
	if lock.RunID != run.ID {
		t.Fatalf("unexpected lock run id")
	}
	if _, err := store.AcquireRunLock(run.ID); err == nil {
		t.Fatalf("expected run lock error")
	}
	stored, err := store.GetRunLock(run.ID)
	if err != nil {
		t.Fatalf("get run lock: %v", err)
	}
	if stored == nil || stored.RunID != run.ID {
		t.Fatalf("expected stored lock")
	}
	if err := store.ReleaseRunLock(run.ID); err != nil {
		t.Fatalf("release run lock: %v", err)
	}
}

func TestListRunsEmpty(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if runs == nil {
		t.Fatalf("expected empty slice, got nil")
	}
	if len(runs) != 0 {
		t.Fatalf("expected zero runs")
	}
	data, err := json.Marshal(runs)
	if err != nil {
		t.Fatalf("marshal runs: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [] payload, got %s", string(data))
	}
}

func TestFinalizeStaleRuns(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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
	old := time.Now().UTC().Add(-10 * time.Minute)
	run.Status = "running"
	run.StartedAt = old
	if err := store.SaveRun(run); err != nil {
		t.Fatalf("save run: %v", err)
	}

	hb := RunHeartbeat{RunID: run.ID, PID: 0, UpdatedAt: old}
	data, err := json.Marshal(hb)
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	if err := os.WriteFile(store.runHeartbeatPath(run.ID), data, 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	lock := RunLock{RunID: run.ID, PID: 0, Hostname: "test", AcquiredAt: old}
	lockData, err := json.Marshal(lock)
	if err != nil {
		t.Fatalf("marshal lock: %v", err)
	}
	if err := os.WriteFile(store.runLockPath(run.ID), lockData, 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	finalized, err := store.FinalizeStaleRuns(60, "stale_after")
	if err != nil {
		t.Fatalf("finalize stale runs: %v", err)
	}
	if len(finalized) != 1 {
		t.Fatalf("expected 1 finalized run, got %d", len(finalized))
	}
	updated, err := store.GetRun(run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status != "failed" || updated.FinishedAt == nil {
		t.Fatalf("expected run failed with finished_at")
	}
	control, err := store.GetRunControl(run.ID)
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if control == nil || control.StopReason == nil {
		t.Fatalf("expected stop reason")
	}
}

func TestCreateSnapshot(t *testing.T) {
	base := t.TempDir()
	store := NewStore(
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
		t.Fatalf("memory docs: %v", err)
	}
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := store.AppendEvent(RunEvent{
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

	snapshot, err := store.CreateSnapshot(run.ID, SnapshotOptions{
		Reason:       "test",
		Sources:      []string{"memory", "events", "artifacts"},
		MaxArtifacts: 10,
		MaxEvents:    10,
		MemoryGlob:   "*.md",
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if snapshot.Summary.ID == "" {
		t.Fatalf("expected snapshot id")
	}
	if snapshot.Manifest.MemoryDocs == 0 {
		t.Fatalf("expected memory docs in snapshot")
	}
	if snapshot.Manifest.Events == 0 {
		t.Fatalf("expected events in snapshot")
	}
	if snapshot.Manifest.Artifacts == 0 {
		t.Fatalf("expected artifacts in snapshot")
	}
	if snapshot.Content == "" {
		t.Fatalf("expected snapshot content")
	}

	summaries, err := store.ListSnapshots(run.ID)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected one snapshot summary")
	}

	loaded, err := store.GetSnapshot(run.ID, snapshot.Summary.ID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if loaded.Summary.ID != snapshot.Summary.ID {
		t.Fatalf("unexpected snapshot id")
	}
}
