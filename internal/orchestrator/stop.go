package orchestrator

import (
	"context"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

func (o *Orchestrator) shouldStop(
	ctx context.Context,
	run *state.Run,
	startTime time.Time,
	lastProgress time.Time,
	consecutiveFailures int,
) (bool, string, string, *string) {
	if ctx.Err() != nil {
		return true, "context_canceled", "canceled", nil
	}

	control, err := o.Store.GetRunControl(run.ID)
	if err == nil && control != nil && control.LastAction != nil {
		action := *control.LastAction
		switch action {
		case "stop", "cancel":
			return true, action, "canceled", &action
		case "kill":
			return true, action, "failed", &action
		}
	}

	if o.Config.Loop.Mode != "continuous" {
		return false, "", "", nil
	}

	if o.Config.Loop.MaxIterations > 0 && run.Iterations >= o.Config.Loop.MaxIterations {
		return true, "max_iterations", "completed", nil
	}
	if o.Config.Loop.StopConditions.MaxDurationSeconds > 0 {
		elapsed := time.Since(startTime)
		if elapsed >= time.Duration(o.Config.Loop.StopConditions.MaxDurationSeconds)*time.Second {
			return true, "max_duration", "completed", nil
		}
	}
	if o.Config.Loop.StopConditions.MaxIdleSeconds > 0 {
		idle := time.Since(lastProgress)
		if idle >= time.Duration(o.Config.Loop.StopConditions.MaxIdleSeconds)*time.Second {
			return true, "max_idle", "canceled", nil
		}
	}
	if o.Config.Loop.StopConditions.MaxConsecutiveFailures > 0 &&
		consecutiveFailures >= o.Config.Loop.StopConditions.MaxConsecutiveFailures {
		return true, "max_consecutive_failures", "failed", nil
	}
	return false, "", "", nil
}

func (o *Orchestrator) finalizeRun(
	run *state.Run,
	status string,
	stopReason *string,
	lastErr error,
	lastAction *string,
) {
	run.Status = status
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	if stopReason != nil {
		run.StopReason = stopReason
	}
	if lastErr != nil {
		msg := lastErr.Error()
		run.LastError = &msg
	}
	if lastAction != nil {
		run.LastAction = lastAction
		run.LastActionAt = &finished
	}
	_ = o.Store.SaveRun(run)

	var errMsg *string
	if lastErr != nil {
		msg := lastErr.Error()
		errMsg = &msg
	}

	_ = o.Store.SaveRunControl(state.RunControl{
		RunID:      run.ID,
		Status:     status,
		StopReason: stopReason,
		LastError:  errMsg,
		LastAction: lastAction,
		UpdatedAt:  time.Now().UTC(),
	})

	if err := o.appendRunSummary(run, stopReason, lastErr, lastAction); err != nil && o.Logger != nil {
		o.Logger.Warn("memory summary append failed", "run_id", run.ID, "error", err.Error())
	}
}
