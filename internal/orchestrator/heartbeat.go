package orchestrator

import (
	"context"
	"time"
)

func (o *Orchestrator) heartbeatLoop(ctx context.Context, runID string, pid int) {
	if o.Store == nil {
		return
	}
	interval := 30 * time.Second
	if o.Config.Loop.StopConditions.MaxHeartbeatSeconds > 0 {
		interval = time.Duration(o.Config.Loop.StopConditions.MaxHeartbeatSeconds/2) * time.Second
		if interval < 15*time.Second {
			interval = 15 * time.Second
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := o.Store.TouchRunHeartbeat(runID, pid); err != nil && o.Logger != nil {
			o.Logger.Warn("heartbeat update failed", "stage", "heartbeat", "run_id", runID, "error", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
