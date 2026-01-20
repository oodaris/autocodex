package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Run struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	CurrentPhase string     `json:"current_phase"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Iterations   int        `json:"iterations"`
}

func (s *Store) CreateRun() (*Run, error) {
	id := fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102T150405Z"), randSuffix(4))
	runDir := filepath.Join(s.RunsDir, id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(runDir, "artifacts"), 0o755); err != nil {
		return nil, fmt.Errorf("create artifacts dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(runDir, "snapshots"), 0o755); err != nil {
		return nil, fmt.Errorf("create snapshots dir: %w", err)
	}
	logPath := s.eventsPath(id)
	if err := os.WriteFile(logPath, []byte(""), 0o644); err != nil {
		return nil, fmt.Errorf("init events log: %w", err)
	}

	run := &Run{
		ID:        id,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	if err := s.SaveRun(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Store) SaveRun(run *Run) error {
	path := filepath.Join(s.RunsDir, run.ID, "run.json")
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) ListRuns() ([]Run, error) {
	entries, err := os.ReadDir(s.RunsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Run{}, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}
	runs := make([]Run, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(s.RunsDir, entry.Name(), "run.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var run Run
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].ID < runs[j].ID
	})
	return runs, nil
}

func (s *Store) GetRun(id string) (Run, error) {
	path := filepath.Join(s.RunsDir, id, "run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Run{}, fmt.Errorf("run not found")
		}
		return Run{}, fmt.Errorf("read run: %w", err)
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		return Run{}, fmt.Errorf("parse run: %w", err)
	}
	return run, nil
}
