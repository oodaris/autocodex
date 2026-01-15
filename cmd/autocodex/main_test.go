package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

func TestRequestRunAction(t *testing.T) {
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
	control, err := requestRunAction(store, run.ID, "kill", "shutdown")
	if err != nil {
		t.Fatalf("request run action: %v", err)
	}
	if control.LastAction == nil || *control.LastAction != "kill" {
		t.Fatalf("expected last action")
	}
	if control.StopReason == nil || *control.StopReason != "shutdown" {
		t.Fatalf("expected stop reason")
	}
}

func TestCollectRunStatuses(t *testing.T) {
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
	now := time.Now().UTC()
	if err := store.SaveRunControl(state.RunControl{
		RunID:        run.ID,
		Status:       "running",
		LastAction:   &action,
		LastActionAt: &now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("save run control: %v", err)
	}
	if err := store.SaveRunFeedback(state.RunFeedback{
		RunID:     run.ID,
		UpdatedAt: now,
		Sources:   []string{"memory"},
	}); err != nil {
		t.Fatalf("save run feedback: %v", err)
	}

	statuses, err := collectRunStatuses(store, run.ID)
	if err != nil {
		t.Fatalf("collect statuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected one status")
	}
	if statuses[0].LastAction == nil || *statuses[0].LastAction != "stop" {
		t.Fatalf("expected last action in status")
	}
	if statuses[0].Feedback == nil || len(statuses[0].Feedback.Sources) != 1 {
		t.Fatalf("expected feedback in status")
	}
}
