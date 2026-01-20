package autonomy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutonomyLoopSmokeTasksSchema(t *testing.T) {
	harness := newSmokeHarness(t, fakeExecutor{}, SmokeHarnessOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := harness.Controller.Run(ctx, Input{Task: "Smoke test autonomy loop schema"}); err != nil {
		t.Fatalf("controller run failed: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(harness.TempDir, "*-tasks.json"))
	if err != nil {
		t.Fatalf("glob tasks: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected tasks json file")
	}

	payload, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read tasks json: %v", err)
	}
	if err := validateJSONSchema(harness.Config.Autonomy.TasksSchema, string(payload)); err != nil {
		t.Fatalf("tasks schema validation failed: %v", err)
	}
	if !strings.Contains(string(payload), `"id": "autocodex-smoke"`) {
		t.Fatalf("expected smoke task in payload")
	}
}
