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

type RunControl struct {
	RunID        string     `json:"run_id"`
	Status       string     `json:"status"`
	StopReason   *string    `json:"stop_reason,omitempty"`
	LastError    *string    `json:"last_error,omitempty"`
	LastAction   *string    `json:"last_action,omitempty"`
	LastActionAt *time.Time `json:"last_action_at,omitempty"`
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
	Bytes             int       `json:"bytes,omitempty"`
}

type RunLock struct {
	RunID      string    `json:"run_id"`
	PID        int       `json:"pid"`
	Hostname   string    `json:"hostname"`
	AcquiredAt time.Time `json:"acquired_at"`
}

var ErrRunLocked = errors.New("run is locked")

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

type MemoryDocSummary struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
}

type MemoryDocDetail struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
	Content   string    `json:"content"`
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

func (s *Store) ListMemoryDocs() ([]MemoryDocSummary, error) {
	entries, err := os.ReadDir(s.MemoryDir)
	if err != nil {
		return nil, fmt.Errorf("read memory dir: %w", err)
	}
	docs := make([]MemoryDocSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat memory doc %s: %w", name, err)
		}
		docs = append(docs, MemoryDocSummary{
			Name:      name,
			Path:      name,
			UpdatedAt: info.ModTime().UTC(),
			SizeBytes: info.Size(),
		})
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Name < docs[j].Name
	})
	return docs, nil
}

func (s *Store) GetMemoryDoc(name string) (*MemoryDocDetail, error) {
	if name == "" {
		return nil, errors.New("memory doc name required")
	}
	if filepath.Base(name) != name || strings.Contains(name, string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid memory doc name: %s", name)
	}
	path := filepath.Join(s.MemoryDir, name)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("stat memory doc: %w", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read memory doc: %w", err)
	}
	return &MemoryDocDetail{
		Name:      name,
		Path:      name,
		UpdatedAt: info.ModTime().UTC(),
		SizeBytes: info.Size(),
		Content:   string(content),
	}, nil
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

func (s *Store) SaveRunControl(control RunControl) error {
	if control.RunID == "" {
		return errors.New("run id required")
	}
	if control.UpdatedAt.IsZero() {
		control.UpdatedAt = time.Now().UTC()
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

func (s *Store) runControlPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "control.json")
}

func (s *Store) runFeedbackPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "feedback.json")
}

func (s *Store) runLockPath(runID string) string {
	return filepath.Join(s.RunsDir, runID, "run.lock")
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
