package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

func (s *Store) eventsPath(runID string) string {
	return filepath.Join(s.LogsDir, runID+".jsonl")
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
