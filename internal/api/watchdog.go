package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

const watchdogInterval = 30 * time.Second

// RunWatchdog finalizes runs that exceed the heartbeat threshold.
type RunWatchdog struct {
	Store               *state.Store
	Logger              *slog.Logger
	MaxHeartbeatSeconds int
}

func (w *RunWatchdog) Start(ctx context.Context) {
	if w.Store == nil || w.MaxHeartbeatSeconds <= 0 {
		return
	}
	ticker := time.NewTicker(watchdogInterval)
	defer ticker.Stop()

	for {
		w.scanOnce()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *RunWatchdog) scanOnce() {
	finalized, err := w.Store.FinalizeStaleRuns(w.MaxHeartbeatSeconds, "watchdog_stale_after")
	if err != nil {
		w.warn("watchdog list runs failed", "error", err.Error())
		return
	}
	for _, run := range finalized {
		w.warn("watchdog marked run failed", "run_id", run.RunID, "reason", run.Reason)
	}
}

func (w *RunWatchdog) warn(msg string, args ...any) {
	if w.Logger != nil {
		w.Logger.Warn(msg, args...)
	}
}
