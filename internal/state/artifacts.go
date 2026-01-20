package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Artifact struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
	Checksum  string    `json:"checksum"`
}

func (s *Store) ListArtifacts(runID string) ([]Artifact, error) {
	dir := filepath.Join(s.RunsDir, runID, "artifacts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Artifact{}, nil
		}
		return nil, fmt.Errorf("read artifacts dir: %w", err)
	}
	var artifacts []Artifact
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		artifact := Artifact{
			ID:        artifactID(runID, name),
			RunID:     runID,
			Name:      name,
			Type:      artifactType(name),
			Path:      filepath.Join(dir, name),
			CreatedAt: info.ModTime().UTC(),
			SizeBytes: info.Size(),
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func (s *Store) GetArtifact(id string) (Artifact, error) {
	runID, name, err := parseArtifactID(id)
	if err != nil {
		return Artifact{}, err
	}
	path := filepath.Join(s.RunsDir, runID, "artifacts", name)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Artifact{}, fmt.Errorf("artifact not found")
		}
		return Artifact{}, fmt.Errorf("stat artifact: %w", err)
	}
	return Artifact{
		ID:        id,
		RunID:     runID,
		Name:      name,
		Type:      artifactType(name),
		Path:      path,
		CreatedAt: info.ModTime().UTC(),
		SizeBytes: info.Size(),
	}, nil
}

func artifactID(runID, name string) string {
	return runID + ":" + name
}

func parseArtifactID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid artifact id")
	}
	return parts[0], parts[1], nil
}

func artifactType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" {
		return strings.TrimPrefix(ext, ".")
	}
	return "file"
}

func isTextArtifact(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".log":
		return true
	default:
		return false
	}
}
