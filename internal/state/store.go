package state

import (
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	StateDir     string
	RunsDir      string
	MemoryDir    string
	LogsDir      string
	ArtifactsDir string
}

func NewStore(stateDir, runsDir, memoryDir, logsDir, artifactsDir string) *Store {
	return &Store{
		StateDir:     stateDir,
		RunsDir:      runsDir,
		MemoryDir:    memoryDir,
		LogsDir:      logsDir,
		ArtifactsDir: artifactsDir,
	}
}

func (s *Store) InitDirs() error {
	for _, dir := range []string{s.StateDir, s.RunsDir, s.MemoryDir, s.LogsDir, s.ArtifactsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("init dir %s: %w", dir, err)
		}
	}
	return nil
}

func (s *Store) eventsPath(runID string) string {
	return filepath.Join(s.LogsDir, runID+".jsonl")
}

func (s *Store) runControlPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "control.json")
}

func (s *Store) runFeedbackPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "feedback.json")
}

func (s *Store) runLockPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "run.lock")
}

func (s *Store) runHeartbeatPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "heartbeat.json")
}

func (s *Store) snapshotsDir(runID string) string {
	return filepath.Join(s.RunsDir, runID, "snapshots")
}

func (s *Store) snapshotRecordPath(runID, snapshotID string) string {
	return filepath.Join(s.snapshotsDir(runID), snapshotID+".json")
}

func (s *Store) snapshotContentPath(runID, snapshotID string) string {
	return filepath.Join(s.snapshotsDir(runID), snapshotID+".md")
}
