package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseSources(t *testing.T) {
	sources := parseSources("memory, events", []string{"artifacts"})
	if len(sources) != 2 || sources[0] != "memory" || sources[1] != "events" {
		t.Fatalf("unexpected sources")
	}
	fallback := parseSources("", []string{"artifacts"})
	if len(fallback) != 1 || fallback[0] != "artifacts" {
		t.Fatalf("expected fallback sources")
	}
}

func TestLatestRunID(t *testing.T) {
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
	if _, err := store.CreateRun(); err != nil {
		t.Fatalf("create run: %v", err)
	}
	id, err := latestRunID(store)
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if id == "" {
		t.Fatalf("expected run id")
	}
}

func TestResolveTaskInputStdin(t *testing.T) {
	payload, err := resolveTaskInput("", "", true, nil, strings.NewReader("  hello world  "))
	if err != nil {
		t.Fatalf("resolve task stdin: %v", err)
	}
	if payload != "hello world" {
		t.Fatalf("expected trimmed payload")
	}
}

func TestResolveTaskInputStdinEmpty(t *testing.T) {
	_, err := resolveTaskInput("", "", true, nil, strings.NewReader("   "))
	if err == nil {
		t.Fatalf("expected error for empty stdin")
	}
}

func TestResolveTaskInputStdinConflict(t *testing.T) {
	_, err := resolveTaskInput("task", "", true, nil, strings.NewReader("ignored"))
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}

func TestResolveTaskInputArgsFallback(t *testing.T) {
	payload, err := resolveTaskInput("", "", false, []string{"hello", "there"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolve task args: %v", err)
	}
	if payload != "hello there" {
		t.Fatalf("expected joined args payload")
	}
}

func TestSelectRunStatusesLatest(t *testing.T) {
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

	firstID := "20260101T000000Z-aaaa"
	secondID := "20260102T000000Z-bbbb"
	createRunWithID(t, store, firstID)
	createRunWithID(t, store, secondID)

	statuses, err := selectRunStatuses(store, "", true)
	if err != nil {
		t.Fatalf("select run statuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected one status")
	}
	if statuses[0].ID != secondID {
		t.Fatalf("expected latest status")
	}
}

func TestSelectRunStatusesLatestConflict(t *testing.T) {
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
	_, err := selectRunStatuses(store, "20260101T000000Z-aaaa", true)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
}

func createRunWithID(t *testing.T, store *state.Store, id string) {
	t.Helper()
	runDir := filepath.Join(store.RunsDir, id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	run := &state.Run{
		ID:        id,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := store.SaveRun(run); err != nil {
		t.Fatalf("save run: %v", err)
	}
}
