package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type RunControl struct {
	RunID        string     `json:"run_id"`
	Status       string     `json:"status"`
	StopReason   *string    `json:"stop_reason,omitempty"`
	LastError    *string    `json:"last_error,omitempty"`
	LastAction   *string    `json:"last_action,omitempty"`
	LastActionAt *time.Time `json:"last_action_at,omitempty"`
	ChildPID     *int       `json:"child_pid,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type RunFeedback struct {
	RunID             string    `json:"run_id"`
	UpdatedAt         time.Time `json:"updated_at"`
	Sources           []string  `json:"sources,omitempty"`
	LastPromptSummary string    `json:"last_prompt_summary,omitempty"`
	LastOutputSummary string    `json:"last_output_summary,omitempty"`
	MemoryDocs        []string  `json:"memory_docs,omitempty"`
	ArtifactIDs       []string  `json:"artifact_ids,omitempty"`
	EventIDs          []string  `json:"event_ids,omitempty"`
	SnapshotPath      string    `json:"snapshot_path,omitempty"`
	Bytes             int       `json:"bytes,omitempty"`
}

type RunLock struct {
	RunID      string    `json:"run_id"`
	PID        int       `json:"pid"`
	Hostname   string    `json:"hostname"`
	AcquiredAt time.Time `json:"acquired_at"`
}

type RunHeartbeat struct {
	RunID     string    `json:"run_id"`
	PID       int       `json:"pid"`
	ChildPID  *int      `json:"child_pid,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

var ErrRunLocked = errors.New("run is locked")

func (s *Store) SaveRunControl(control RunControl) error {
	if control.RunID == "" {
		return errors.New("run id required")
	}
	if control.UpdatedAt.IsZero() {
		control.UpdatedAt = time.Now().UTC()
	}
	if control.ChildPID == nil {
		if existing, _ := s.GetRunControl(control.RunID); existing != nil && existing.ChildPID != nil && *existing.ChildPID > 0 {
			control.ChildPID = existing.ChildPID
		}
	}
	path := s.runControlPath(control.RunID)
	data, err := json.MarshalIndent(control, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run control: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) GetRunControl(runID string) (*RunControl, error) {
	path := s.runControlPath(runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read run control: %w", err)
	}
	var control RunControl
	if err := json.Unmarshal(data, &control); err != nil {
		return nil, fmt.Errorf("parse run control: %w", err)
	}
	return &control, nil
}

func (s *Store) SetRunChildPID(runID string, childPID int) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id required")
	}
	if childPID <= 0 {
		return nil
	}
	control, err := s.GetRunControl(runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if control == nil {
		control = &RunControl{
			RunID:     runID,
			Status:    "running",
			UpdatedAt: now,
		}
	}
	control.ChildPID = &childPID
	control.UpdatedAt = now
	if err := s.SaveRunControl(*control); err != nil {
		return err
	}
	return s.setHeartbeatChildPID(runID, &childPID)
}

func (s *Store) ClearRunChildPID(runID string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id required")
	}
	control, err := s.GetRunControl(runID)
	if err != nil || control == nil {
		return err
	}
	if control.ChildPID == nil {
		return s.setHeartbeatChildPID(runID, nil)
	}
	zero := 0
	control.ChildPID = &zero
	control.UpdatedAt = time.Now().UTC()
	if err := s.SaveRunControl(*control); err != nil {
		return err
	}
	return s.setHeartbeatChildPID(runID, nil)
}

func (s *Store) SaveRunFeedback(feedback RunFeedback) error {
	if feedback.RunID == "" {
		return errors.New("run id required")
	}
	if feedback.UpdatedAt.IsZero() {
		feedback.UpdatedAt = time.Now().UTC()
	}
	path := s.runFeedbackPath(feedback.RunID)
	data, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run feedback: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) GetRunFeedback(runID string) (*RunFeedback, error) {
	path := s.runFeedbackPath(runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read run feedback: %w", err)
	}
	var feedback RunFeedback
	if err := json.Unmarshal(data, &feedback); err != nil {
		return nil, fmt.Errorf("parse run feedback: %w", err)
	}
	return &feedback, nil
}

func (s *Store) AcquireRunLock(runID string) (*RunLock, error) {
	if runID == "" {
		return nil, errors.New("run id required")
	}
	path := s.runLockPath(runID)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrRunLocked
		}
		return nil, fmt.Errorf("create run lock: %w", err)
	}
	defer file.Close()

	host, _ := os.Hostname()
	lock := &RunLock{
		RunID:      runID,
		PID:        os.Getpid(),
		Hostname:   host,
		AcquiredAt: time.Now().UTC(),
	}
	data, err := json.Marshal(lock)
	if err != nil {
		return nil, fmt.Errorf("marshal run lock: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return nil, fmt.Errorf("write run lock: %w", err)
	}
	return lock, nil
}

func (s *Store) GetRunLock(runID string) (*RunLock, error) {
	path := s.runLockPath(runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read run lock: %w", err)
	}
	var lock RunLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parse run lock: %w", err)
	}
	return &lock, nil
}

func (s *Store) ReleaseRunLock(runID string) error {
	path := s.runLockPath(runID)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove run lock: %w", err)
	}
	return nil
}

func (s *Store) TouchRunHeartbeat(runID string, pid int) error {
	if runID == "" {
		return errors.New("run id required")
	}
	path := s.runHeartbeatPath(runID)
	var childPID *int
	if existing, err := s.GetRunHeartbeat(runID); err == nil && existing != nil {
		if existing.ChildPID != nil && *existing.ChildPID > 0 {
			childPID = existing.ChildPID
		}
	}
	hb := RunHeartbeat{
		RunID:     runID,
		PID:       pid,
		ChildPID:  childPID,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("marshal run heartbeat: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) GetRunHeartbeat(runID string) (*RunHeartbeat, error) {
	path := s.runHeartbeatPath(runID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read run heartbeat: %w", err)
	}
	var hb RunHeartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return nil, fmt.Errorf("parse run heartbeat: %w", err)
	}
	return &hb, nil
}

func (s *Store) setHeartbeatChildPID(runID string, childPID *int) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id required")
	}
	hb, err := s.GetRunHeartbeat(runID)
	if err != nil {
		return err
	}
	if hb == nil {
		hb = &RunHeartbeat{
			RunID: runID,
		}
	}
	if childPID != nil && *childPID <= 0 {
		childPID = nil
	}
	hb.ChildPID = childPID
	hb.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("marshal run heartbeat: %w", err)
	}
	return os.WriteFile(s.runHeartbeatPath(runID), data, 0o644)
}
