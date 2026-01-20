package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		runDir := filepath.Join(s.RunsDir, run.ID)
		if err := os.RemoveAll(runDir); err != nil {
			return result, fmt.Errorf("remove run dir %s: %w", runDir, err)
		}
		logPath := filepath.Join(s.LogsDir, run.ID+".jsonl")
		if err := os.Remove(logPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("remove run log %s: %w", logPath, err)
		}
	}
	return result, nil
}
