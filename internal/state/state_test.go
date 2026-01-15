package state

import (
	"os"
	"path/filepath"
	"testing"
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
