package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type AutonomyArtifacts struct {
	Tag       string    `json:"tag"`
	SpecPath  string    `json:"spec_path"`
	PlanPath  string    `json:"plan_path"`
	TasksPath string    `json:"tasks_path"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) SaveAutonomyArtifacts(art AutonomyArtifacts) error {
	if art.Tag == "" {
		return errors.New("autonomy artifacts tag is required")
	}
	if art.CreatedAt.IsZero() {
		art.CreatedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(s.autonomyArtifactsDir(), 0o755); err != nil {
		return fmt.Errorf("init autonomy artifacts dir: %w", err)
	}
	data, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal autonomy artifacts: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.autonomyArtifactsDir(), "latest.json"), data, 0o644); err != nil {
		return fmt.Errorf("write autonomy artifacts latest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.autonomyArtifactsDir(), art.Tag+".json"), data, 0o644); err != nil {
		return fmt.Errorf("write autonomy artifacts: %w", err)
	}
	return nil
}

func (s *Store) LoadLatestAutonomyArtifacts() (*AutonomyArtifacts, error) {
	path := filepath.Join(s.autonomyArtifactsDir(), "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read autonomy artifacts: %w", err)
	}
	var art AutonomyArtifacts
	if err := json.Unmarshal(data, &art); err != nil {
		return nil, fmt.Errorf("parse autonomy artifacts: %w", err)
	}
	return &art, nil
}

func (s *Store) autonomyArtifactsDir() string {
	return filepath.Join(s.ArtifactsDir, "autonomy")
}
