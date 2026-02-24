package autonomy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildBeadScopeFromTasks(t *testing.T) {
	scope := buildBeadScopeFromTasks([]Task{
		{ID: "integration-a001"},
		{ID: "integration-a002"},
	})
	if !scope["integration-a001"] || !scope["integration-a002"] {
		t.Fatalf("expected task ids in scope, got %#v", scope)
	}
}

func TestBuildBeadScopeFromTasksPath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks.json")
	payload := `{"version":"1.0","generated_at":"2026-01-01T00:00:00Z","source_plan":"plan.md","tasks":[{"id":"integration-a001","title":"A","goal":"g","scope":[],"files":[],"dependencies":[],"skills":[],"acceptance_criteria":[],"tests":[],"docs":[],"rollout":"","observability":"","status":"todo","priority":0,"owner":"","notes":""}]}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write tasks: %v", err)
	}
	scope, err := buildBeadScopeFromTasksPath(path)
	if err != nil {
		t.Fatalf("build scope from tasks path: %v", err)
	}
	if !scope["integration-a001"] {
		t.Fatalf("expected integration-a001 in scope, got %#v", scope)
	}
}

func TestFilterReadyBeadsBySelectors(t *testing.T) {
	beads := []ReadyBead{
		{ID: "integration-a001"},
		{ID: "integration-b001"},
		{ID: "integration-c001"},
	}
	filtered := filterReadyBeadsBySelectors(beads, []string{"integration-a001", "integration-c001"}, "")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered beads, got %d", len(filtered))
	}
	prefixFiltered := filterReadyBeadsBySelectors(beads, nil, "integration-b")
	if len(prefixFiltered) != 1 || prefixFiltered[0].ID != "integration-b001" {
		t.Fatalf("unexpected prefix filtered beads: %#v", prefixFiltered)
	}
}

func TestListReadyBeadsWithScopeFailClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based bd fake not supported on windows")
	}
	tmp := t.TempDir()
	bdBin := filepath.Join(tmp, "bd")
	if err := writeBDFake(bdBin); err != nil {
		t.Fatalf("write fake bd: %v", err)
	}
	bdState := filepath.Join(tmp, "bd-state")
	if err := os.WriteFile(bdState, []byte("integration-x|todo|X|0\n"), 0o644); err != nil {
		t.Fatalf("write bd state: %v", err)
	}
	t.Setenv("PATH", fmt.Sprintf("%s:%s", tmp, os.Getenv("PATH")))
	t.Setenv("BD_STATE", bdState)

	_, _, err := listReadyBeadsWithScope(map[string]bool{"integration-a": true}, "run_scoped", false)
	if err == nil {
		t.Fatalf("expected fail-closed scope error")
	}
	if !errors.Is(err, ErrNoRunScopedReadyBeads) {
		t.Fatalf("expected ErrNoRunScopedReadyBeads, got %v", err)
	}
}
