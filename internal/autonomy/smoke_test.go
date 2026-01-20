package autonomy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/codex"
)

type fakeExecutor struct{}

func (fakeExecutor) Exec(_ context.Context, prompt string) (codex.ExecResult, error) {
	result := codex.ExecResult{}
	switch {
	case strings.Contains(prompt, "Schema (MUST conform)") && strings.Contains(prompt, "Tasks Schema"):
		result.Stdout = `{"version":"1.0","generated_at":"2026-01-15T00:00:00Z","source_plan":"plan.md","tasks":[{"id":"autocodex-smoke","title":"Smoke bead","goal":"Verify autonomy loop","files":[],"acceptance_criteria":[]}]}`
		return result, nil
	case strings.Contains(prompt, "Template (fill in"):
		result.Stdout = "# Spec\n\nok\n"
		return result, nil
	case strings.Contains(prompt, "Phase: test"):
		result.Stdout = "tests ok\nACTIONS_JSON_START\n{\"version\":\"1.0\",\"summary\":\"done\",\"next\":{\"type\":\"none\"},\"updates\":{\"beads\":[{\"id\":\"autocodex-smoke\",\"status\":\"done\"}]},\"gates\":{\"review_required\":false,\"tests\":[],\"blocking\":false},\"stop\":null}\nACTIONS_JSON_END\n"
		return result, nil
	default:
		result.Stdout = "ok\n"
		return result, nil
	}
}

func TestAutonomyLoopSmoke(t *testing.T) {
	harness := newSmokeHarness(t, fakeExecutor{}, SmokeHarnessOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := harness.Controller.Run(ctx, Input{Task: "Smoke test autonomy loop"}); err != nil {
		t.Fatalf("controller run failed: %v", err)
	}

	if !strings.Contains(harness.ReadBDState(t), "autocodex-smoke|done") {
		t.Fatalf("expected bead to be marked done")
	}
}
