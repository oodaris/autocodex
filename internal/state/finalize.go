package state

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type FinalizedRun struct {
	RunID  string
	Reason string
}

func (s *Store) FinalizeRun(runID, status, reason, action string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id required")
	}
	run, err := s.GetRun(runID)
	if err != nil {
		return err
	}
	if status == "" {
		status = "failed"
	}
	now := time.Now().UTC()
	run.Status = status
	run.FinishedAt = &now
	if err := s.SaveRun(&run); err != nil {
		return err
	}
	control := RunControl{
		RunID:     run.ID,
		Status:    status,
		UpdatedAt: now,
	}
	if strings.TrimSpace(reason) != "" {
		control.StopReason = &reason
	}
	if strings.TrimSpace(action) != "" {
		control.LastAction = &action
		control.LastActionAt = &now
	}
	if err := s.SaveRunControl(control); err != nil {
		return err
	}
	_ = s.ReleaseRunLock(run.ID)
	_ = s.AppendEvent(RunEvent{
		ID:      fmt.Sprintf("run-stopped-%d", time.Now().UnixNano()),
		RunID:   run.ID,
		TS:      now,
		Type:    "run_stopped",
		Message: reason,
		Meta:    map[string]string{"auto_finalizer": "true"},
	})
	return nil
}

func (s *Store) FinalizeStaleRuns(maxHeartbeatSeconds int, reasonPrefix string) ([]FinalizedRun, error) {
	if maxHeartbeatSeconds <= 0 {
		return nil, nil
	}
	runs, err := s.ListRuns()
	if err != nil {
		return nil, err
	}
	threshold := time.Duration(maxHeartbeatSeconds) * time.Second
	now := time.Now().UTC()
	results := make([]FinalizedRun, 0)

	for _, run := range runs {
		if run.Status != "running" {
			continue
		}

		lock, _ := s.GetRunLock(run.ID)
		heartbeat, _ := s.GetRunHeartbeat(run.ID)
		control, _ := s.GetRunControl(run.ID)

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
			pidAlive = IsProcessAlive(lock.PID)
		}
		childAlive := false
		if control != nil && control.ChildPID != nil && *control.ChildPID > 0 {
			childAlive = IsProcessAlive(*control.ChildPID)
		}
		if heartbeat != nil && heartbeat.ChildPID != nil && *heartbeat.ChildPID > 0 {
			childAlive = childAlive || IsProcessAlive(*heartbeat.ChildPID)
		}
		if pidAlive || childAlive {
			continue
		}

		prefix := "stale_after"
		if strings.TrimSpace(reasonPrefix) != "" {
			prefix = strings.TrimSpace(reasonPrefix)
		}
		reason := fmt.Sprintf("%s_%ds", prefix, maxHeartbeatSeconds)
		if err := s.FinalizeRun(run.ID, "failed", reason, "stale"); err != nil {
			return results, err
		}
		results = append(results, FinalizedRun{RunID: run.ID, Reason: reason})
	}
	return results, nil
}
