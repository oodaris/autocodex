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

type SnapshotSummary struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	CreatedAt   time.Time `json:"created_at"`
	Reason      string    `json:"reason"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentPath string    `json:"content_path"`
}

type SnapshotManifest struct {
	Events     int `json:"events"`
	Artifacts  int `json:"artifacts"`
	MemoryDocs int `json:"memory_docs"`
	Bytes      int `json:"bytes"`
}

type SnapshotDetail struct {
	Summary  SnapshotSummary  `json:"summary"`
	Manifest SnapshotManifest `json:"manifest"`
	Content  string           `json:"content"`
}

type SnapshotOptions struct {
	Reason       string
	Sources      []string
	MaxBytes     int
	MaxEvents    int
	MaxArtifacts int
	MemoryGlob   string
}

type CleanupOptions struct {
	OlderThan time.Duration
	DryRun    bool
	Now       time.Time
}

type CleanupResult struct {
	Deleted []string
	Skipped []string
}

type FinalizedRun struct {
	RunID  string
	Reason string
}

type snapshotRecord struct {
	Summary  SnapshotSummary  `json:"summary"`
	Manifest SnapshotManifest `json:"manifest"`
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

func (s *Store) AppendMemoryDoc(name, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	if name == "" {
		return errors.New("memory doc name required")
	}
	if filepath.Base(name) != name || strings.Contains(name, string(filepath.Separator)) {
		return fmt.Errorf("invalid memory doc name: %s", name)
	}
	if err := os.MkdirAll(s.MemoryDir, 0o755); err != nil {
		return fmt.Errorf("init memory dir: %w", err)
	}
	path := filepath.Join(s.MemoryDir, name)
	payload := strings.TrimRight(content, "\n")
	payload = "\n" + payload + "\n"
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open memory doc: %w", err)
	}
	defer file.Close()
	if _, err := file.WriteString(payload); err != nil {
		return fmt.Errorf("append memory doc: %w", err)
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

func (s *Store) FinalizeRun(runID, status, reason, action string) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id required")
	}
	run, err := s.GetRun(runID)
	if err != nil {
		return err
	}
	if status == "" {
		status = "failed"
	}
	now := time.Now().UTC()
	run.Status = status
	run.FinishedAt = &now
	if err := s.SaveRun(&run); err != nil {
		return err
	}
	control := RunControl{
		RunID:     run.ID,
		Status:    status,
		UpdatedAt: now,
	}
	if strings.TrimSpace(reason) != "" {
		control.StopReason = &reason
	}
	if strings.TrimSpace(action) != "" {
		control.LastAction = &action
		control.LastActionAt = &now
	}
	if err := s.SaveRunControl(control); err != nil {
		return err
	}
	_ = s.ReleaseRunLock(run.ID)
	_ = s.AppendEvent(RunEvent{
		ID:      fmt.Sprintf("run-stopped-%d", time.Now().UnixNano()),
		RunID:   run.ID,
		TS:      now,
		Type:    "run_stopped",
		Message: reason,
		Meta:    map[string]string{"auto_finalizer": "true"},
	})
	return nil
}

func (s *Store) FinalizeStaleRuns(maxHeartbeatSeconds int, reasonPrefix string) ([]FinalizedRun, error) {
	if maxHeartbeatSeconds <= 0 {
		return nil, nil
	}
	runs, err := s.ListRuns()
	if err != nil {
		return nil, err
	}
	threshold := time.Duration(maxHeartbeatSeconds) * time.Second
	now := time.Now().UTC()
	results := make([]FinalizedRun, 0)

	for _, run := range runs {
		if run.Status != "running" {
			continue
		}

		lock, _ := s.GetRunLock(run.ID)
		heartbeat, _ := s.GetRunHeartbeat(run.ID)
		control, _ := s.GetRunControl(run.ID)

		lastSeen := run.StartedAt
		if control != nil && control.UpdatedAt.After(lastSeen) {
			lastSeen = control.UpdatedAt
		}
		if heartbeat != nil && heartbeat.UpdatedAt.After(lastSeen) {
			lastSeen = heartbeat.UpdatedAt
		}

		if now.Sub(lastSeen) < threshold {
			continue
		}

		pidAlive := false
		if lock != nil && lock.PID > 0 {
			pidAlive = IsProcessAlive(lock.PID)
		}
		childAlive := false
		if control != nil && control.ChildPID != nil && *control.ChildPID > 0 {
			childAlive = IsProcessAlive(*control.ChildPID)
		}
		if heartbeat != nil && heartbeat.ChildPID != nil && *heartbeat.ChildPID > 0 {
			childAlive = childAlive || IsProcessAlive(*heartbeat.ChildPID)
		}
		if pidAlive || childAlive {
			continue
		}

		prefix := "stale_after"
		if strings.TrimSpace(reasonPrefix) != "" {
			prefix = strings.TrimSpace(reasonPrefix)
		}
		reason := fmt.Sprintf("%s_%ds", prefix, maxHeartbeatSeconds)
		if err := s.FinalizeRun(run.ID, "failed", reason, "stale"); err != nil {
			return results, err
		}
		results = append(results, FinalizedRun{RunID: run.ID, Reason: reason})
	}
	return results, nil
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

func (s *Store) CreateSnapshot(runID string, opts SnapshotOptions) (SnapshotDetail, error) {
	if runID == "" {
		return SnapshotDetail{}, errors.New("run id required")
	}
	if _, err := s.GetRun(runID); err != nil {
		return SnapshotDetail{}, err
	}

	if opts.MaxBytes < 0 {
		return SnapshotDetail{}, errors.New("max bytes must be >= 0")
	}
	if opts.MaxEvents < 0 || opts.MaxArtifacts < 0 {
		return SnapshotDetail{}, errors.New("max events/artifacts must be >= 0")
	}
	if len(opts.Sources) == 0 {
		opts.Sources = []string{"memory", "events", "artifacts"}
	}
	if opts.MemoryGlob == "" {
		opts.MemoryGlob = "*.md"
	}

	createdAt := time.Now().UTC()
	snapshotID := fmt.Sprintf("%s-%s", createdAt.Format("20060102T150405Z"), randSuffix(3))
	snapshotsDir := s.snapshotsDir(runID)
	if err := os.MkdirAll(snapshotsDir, 0o755); err != nil {
		return SnapshotDetail{}, fmt.Errorf("create snapshots dir: %w", err)
	}
	contentPath := s.snapshotContentPath(runID, snapshotID)
	recordPath := s.snapshotRecordPath(runID, snapshotID)

	var b strings.Builder
	appendWithLimit := newLimitWriter(&b, opts.MaxBytes)
	appendWithLimit(fmt.Sprintf("# autocodex Snapshot\nRun: %s\nCreated: %s\n", runID, createdAt.Format(time.RFC3339)))
	if opts.Reason != "" {
		appendWithLimit(fmt.Sprintf("Reason: %s\n", opts.Reason))
	}
	appendWithLimit("\n")

	manifest := SnapshotManifest{}

	if sourceEnabled(opts.Sources, "memory") {
		docs, err := s.ListMemoryDocs()
		if err != nil {
			return SnapshotDetail{}, err
		}
		for _, doc := range docs {
			matched, err := filepath.Match(opts.MemoryGlob, doc.Name)
			if err != nil || !matched {
				continue
			}
			detail, err := s.GetMemoryDoc(doc.Name)
			if err != nil {
				continue
			}
			manifest.MemoryDocs++
			if !appendWithLimit(fmt.Sprintf("## Memory: %s\n%s\n\n", detail.Name, detail.Content)) {
				break
			}
		}
	}

	if sourceEnabled(opts.Sources, "events") {
		events, err := s.ListEvents(runID)
		if err != nil {
			return SnapshotDetail{}, err
		}
		sort.Slice(events, func(i, j int) bool {
			if events[i].TS.Equal(events[j].TS) {
				return events[i].ID < events[j].ID
			}
			return events[i].TS.Before(events[j].TS)
		})
		if opts.MaxEvents > 0 && len(events) > opts.MaxEvents {
			events = events[len(events)-opts.MaxEvents:]
		}
		if len(events) > 0 {
			appendWithLimit("## Recent Events\n")
			for _, event := range events {
				manifest.Events++
				line := fmt.Sprintf("- %s [%s] %s %s\n", event.TS.Format(time.RFC3339), event.Type, event.Phase, event.Message)
				if !appendWithLimit(line) {
					break
				}
			}
			appendWithLimit("\n")
		}
	}

	if sourceEnabled(opts.Sources, "artifacts") {
		artifacts, err := s.ListArtifacts(runID)
		if err != nil {
			return SnapshotDetail{}, err
		}
		sort.Slice(artifacts, func(i, j int) bool {
			if artifacts[i].CreatedAt.Equal(artifacts[j].CreatedAt) {
				return artifacts[i].Name < artifacts[j].Name
			}
			return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
		})
		if opts.MaxArtifacts > 0 && len(artifacts) > opts.MaxArtifacts {
			artifacts = artifacts[len(artifacts)-opts.MaxArtifacts:]
		}
		if len(artifacts) > 0 {
			appendWithLimit("## Recent Artifacts\n")
			for _, artifact := range artifacts {
				manifest.Artifacts++
				line := fmt.Sprintf("- %s (%s, %d bytes)\n", artifact.Name, artifact.Type, artifact.SizeBytes)
				if !appendWithLimit(line) {
					break
				}
			}
			appendWithLimit("\n")
		}
		for _, artifact := range artifacts {
			if !isTextArtifact(artifact.Name) {
				continue
			}
			content, err := os.ReadFile(artifact.Path)
			if err != nil {
				continue
			}
			if !appendWithLimit(fmt.Sprintf("### Artifact: %s\n", artifact.Name)) {
				break
			}
			if !appendWithLimit("```\n") {
				break
			}
			if !appendWithLimit(string(content)) {
				appendWithLimit("\n```\n")
				break
			}
			appendWithLimit("\n```\n")
		}
	}

	content := b.String()
	if err := os.WriteFile(contentPath, []byte(content), 0o644); err != nil {
		return SnapshotDetail{}, fmt.Errorf("write snapshot content: %w", err)
	}

	summary := SnapshotSummary{
		ID:          snapshotID,
		RunID:       runID,
		CreatedAt:   createdAt,
		Reason:      opts.Reason,
		SizeBytes:   int64(len(content)),
		ContentPath: contentPath,
	}
	manifest.Bytes = len(content)
	record := snapshotRecord{Summary: summary, Manifest: manifest}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return SnapshotDetail{}, fmt.Errorf("marshal snapshot record: %w", err)
	}
	if err := os.WriteFile(recordPath, data, 0o644); err != nil {
		return SnapshotDetail{}, fmt.Errorf("write snapshot record: %w", err)
	}
	return SnapshotDetail{
		Summary:  summary,
		Manifest: manifest,
		Content:  content,
	}, nil
}

func (s *Store) ListSnapshots(runID string) ([]SnapshotSummary, error) {
	dir := s.snapshotsDir(runID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SnapshotSummary{}, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}
	summaries := make([]SnapshotSummary, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var record snapshotRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		summaries = append(summaries, record.Summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt.Before(summaries[j].CreatedAt)
	})
	return summaries, nil
}

func (s *Store) GetSnapshot(runID, snapshotID string) (SnapshotDetail, error) {
	if runID == "" || snapshotID == "" {
		return SnapshotDetail{}, errors.New("run id and snapshot id required")
	}
	recordPath := s.snapshotRecordPath(runID, snapshotID)
	data, err := os.ReadFile(recordPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SnapshotDetail{}, fmt.Errorf("snapshot not found")
		}
		return SnapshotDetail{}, fmt.Errorf("read snapshot record: %w", err)
	}
	var record snapshotRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return SnapshotDetail{}, fmt.Errorf("parse snapshot record: %w", err)
	}
	content, err := os.ReadFile(record.Summary.ContentPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return SnapshotDetail{}, fmt.Errorf("read snapshot content: %w", err)
	}
	return SnapshotDetail{
		Summary:  record.Summary,
		Manifest: record.Manifest,
		Content:  string(content),
	}, nil
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

func isTextArtifact(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".log":
		return true
	default:
		return false
	}
}

func trimExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

func sourceEnabled(sources []string, name string) bool {
	for _, source := range sources {
		if source == name {
			return true
		}
	}
	return false
}

func newLimitWriter(builder *strings.Builder, maxBytes int) func(text string) bool {
	if maxBytes <= 0 {
		return func(text string) bool {
			if text == "" {
				return true
			}
			builder.WriteString(text)
			return true
		}
	}
	remaining := maxBytes
	return func(text string) bool {
		if text == "" {
			return true
		}
		if remaining <= 0 {
			return false
		}
		if len(text) > remaining {
			text = text[:remaining]
		}
		builder.WriteString(text)
		remaining -= len(text)
		return remaining > 0
	}
}

func randSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
