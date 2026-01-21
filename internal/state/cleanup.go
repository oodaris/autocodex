package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CleanupOptions struct {
	OlderThan time.Duration
	DryRun    bool
	Now       time.Time
}

type CleanupResult struct {
	Deleted []string
	Skipped []string
}

func (s *Store) DeleteRun(runID string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id is required")
	}
	runDir := filepath.Join(s.RunsDir, runID)
	if _, err := os.Stat(runDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("run not found")
		}
		return fmt.Errorf("stat run dir %s: %w", runDir, err)
	}
	if err := os.RemoveAll(runDir); err != nil {
		return fmt.Errorf("remove run dir %s: %w", runDir, err)
	}
	logPath := filepath.Join(s.LogsDir, runID+".jsonl")
	if err := os.Remove(logPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove run log %s: %w", logPath, err)
	}
	return nil
}

func (s *Store) CleanupRuns(opts CleanupOptions) (CleanupResult, error) {
	result := CleanupResult{}
	if opts.OlderThan < 0 {
		return result, errors.New("older than must be >= 0")
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runs, err := s.ListRuns()
	if err != nil {
		return result, err
	}
	if len(runs) == 0 {
		return result, nil
	}

	cutoff := now.Add(-opts.OlderThan)
	for _, run := range runs {
		if run.FinishedAt == nil {
			result.Skipped = append(result.Skipped, run.ID)
			continue
		}
		if run.FinishedAt.After(cutoff) {
			result.Skipped = append(result.Skipped, run.ID)
			continue
		}
		result.Deleted = append(result.Deleted, run.ID)
		if opts.DryRun {
			continue
		}
		if err := s.DeleteRun(run.ID); err != nil {
			return result, err
		}
	}
	return result, nil
}
