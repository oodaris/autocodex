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
