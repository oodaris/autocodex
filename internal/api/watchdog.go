package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

const watchdogInterval = 30 * time.Second

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
	runs, err := w.Store.ListRuns()
	if err != nil {
		w.warn("watchdog list runs failed", "error", err.Error())
		return
	}
	threshold := time.Duration(w.MaxHeartbeatSeconds) * time.Second
	now := time.Now().UTC()

	for _, run := range runs {
		if run.Status != "running" {
			continue
		}

		lock, _ := w.Store.GetRunLock(run.ID)
		heartbeat, _ := w.Store.GetRunHeartbeat(run.ID)
		control, _ := w.Store.GetRunControl(run.ID)

		lastSeen := run.StartedAt
		if control != nil && control.UpdatedAt.After(lastSeen) {
			lastSeen = control.UpdatedAt
		}
		if heartbeat != nil && heartbeat.UpdatedAt.After(lastSeen) {
			lastSeen = heartbeat.UpdatedAt
		}

		if now.Sub(lastSeen) < threshold {
			continue
		}

		pidAlive := false
		if lock != nil && lock.PID > 0 {
			pidAlive = state.IsProcessAlive(lock.PID)
		}
		if pidAlive {
			w.warn("watchdog heartbeat stale but pid alive", "run_id", run.ID, "pid", lock.PID)
			continue
		}

		reason := fmt.Sprintf("watchdog_stale_after_%ds", w.MaxHeartbeatSeconds)
		finished := now
		run.Status = "failed"
		run.FinishedAt = &finished
		if err := w.Store.SaveRun(&run); err != nil {
			w.warn("watchdog save run failed", "run_id", run.ID, "error", err.Error())
			continue
		}
		controlUpdate := state.RunControl{
			RunID:      run.ID,
			Status:     "failed",
			StopReason: &reason,
			UpdatedAt:  now,
		}
		if err := w.Store.SaveRunControl(controlUpdate); err != nil {
			w.warn("watchdog save control failed", "run_id", run.ID, "error", err.Error())
		}
		_ = w.Store.ReleaseRunLock(run.ID)
		_ = w.Store.AppendEvent(state.RunEvent{
			ID:      fmt.Sprintf("watchdog-%d", time.Now().UnixNano()),
			RunID:   run.ID,
			TS:      now,
			Type:    "run_stopped",
			Message: reason,
			Meta:    map[string]string{"watchdog": "true"},
		})

		w.warn("watchdog marked run failed", "run_id", run.ID, "reason", reason)
	}
}

func (w *RunWatchdog) warn(msg string, args ...any) {
	if w.Logger != nil {
		w.Logger.Warn(msg, args...)
	}
}
