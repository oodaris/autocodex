package state

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct {
	StateDir     string
	RunsDir      string
	MemoryDir    string
	LogsDir      string
	ArtifactsDir string
}

type Run struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	CurrentPhase string     `json:"current_phase"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Iterations   int        `json:"iterations"`
}

type RunEvent struct {
	ID      string            `json:"id"`
	RunID   string            `json:"run_id"`
	TS      time.Time         `json:"ts"`
	Type    string            `json:"type"`
	Phase   string            `json:"phase"`
	Message string            `json:"message"`
	Meta    map[string]string `json:"meta"`
}

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

func (s *Store) EnsureMemoryDocs() error {
	docs := []string{"TODO.md", "PROGRESS.md", "OPINIONS.md", "SPEC.md"}
	for _, name := range docs {
		path := filepath.Join(s.MemoryDir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte("# "+trimExt(name)+"\n"), 0o644); err != nil {
			return fmt.Errorf("write memory doc %s: %w", path, err)
		}
	}
	return nil
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

func (s *Store) AppendEvent(event RunEvent) error {
	path := s.eventsPath(event.RunID)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open events log: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func (s *Store) ListRuns() ([]Run, error) {
	entries, err := os.ReadDir(s.RunsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Run{}, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}
	var runs []Run
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

func (s *Store) ListEvents(runID string) ([]RunEvent, error) {
	path := s.eventsPath(runID)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []RunEvent{}, nil
		}
		return nil, fmt.Errorf("open events: %w", err)
	}
	defer file.Close()

	return decodeEvents(file)
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

func (s *Store) eventsPath(runID string) string {
	return filepath.Join(s.LogsDir, runID+".jsonl")
}

func decodeEvents(r io.Reader) ([]RunEvent, error) {
	scanner := bufio.NewScanner(r)
	var events []RunEvent
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event RunEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, scanner.Err()
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

func trimExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

func randSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
