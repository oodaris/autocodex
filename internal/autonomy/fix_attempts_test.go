package autonomy

import (
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestFixAttemptTrackerPersistsWhenGlobal(t *testing.T) {
	tmp := t.TempDir()
	storePath := filepath.Join(tmp, "fix_attempts.json")

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Autonomy.Gate.MaxFixAttemptsScope = "global"
	cfg.Autonomy.Gate.FixAttemptsStore = storePath

	first := newFixAttemptTracker(cfg, nil)
	if got := first.Increment("integration-a001"); got != 1 {
		t.Fatalf("expected first increment to be 1, got %d", got)
	}

	second := newFixAttemptTracker(cfg, nil)
	if got := second.Increment("integration-a001"); got != 2 {
		t.Fatalf("expected persisted increment to be 2, got %d", got)
	}
	second.Reset("integration-a001")
	third := newFixAttemptTracker(cfg, nil)
	if got := third.Increment("integration-a001"); got != 1 {
		t.Fatalf("expected reset to persist, got %d", got)
	}
}

func TestFixAttemptTrackerRunScopeDoesNotPersist(t *testing.T) {
	tmp := t.TempDir()
	storePath := filepath.Join(tmp, "fix_attempts.json")

	cfg := config.Config{Version: "v1", Mode: "yolo"}
	cfg.ApplyDefaults()
	cfg.Autonomy.Gate.MaxFixAttemptsScope = "run"
	cfg.Autonomy.Gate.FixAttemptsStore = storePath

	first := newFixAttemptTracker(cfg, nil)
	if got := first.Increment("integration-a001"); got != 1 {
		t.Fatalf("expected increment to be 1, got %d", got)
	}
	second := newFixAttemptTracker(cfg, nil)
	if got := second.Increment("integration-a001"); got != 1 {
		t.Fatalf("expected run-scoped tracker to reset between runs, got %d", got)
	}
}
