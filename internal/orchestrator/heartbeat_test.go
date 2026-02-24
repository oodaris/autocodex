package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/state"
)

func TestHeartbeatLoopSkipsWriteWhenCanceled(t *testing.T) {
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	orch := Orchestrator{Store: store}
	orch.heartbeatLoop(ctx, run.ID, os.Getpid())

	heartbeatPath := filepath.Join(store.RunsDir, run.ID, "heartbeat.json")
	if _, err := os.Stat(heartbeatPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no heartbeat write after cancel, got err=%v", err)
	}
}
