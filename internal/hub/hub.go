package hub

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

const (
	StatusOK            = "ok"
	StatusConfigMissing = "missing_config"
	StatusConfigInvalid = "invalid_config"
	StatusStateError    = "state_error"
)

type Workspace struct {
	ID         string
	Name       string
	Root       string
	ConfigPath string
	Config     *config.Config
	Store      *state.Store
	Err        error
}

type WorkspaceSummary struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Root            string  `json:"root"`
	ConfigPath      string  `json:"config_path"`
	Status          string  `json:"status"`
	Error           *string `json:"error,omitempty"`
	RunsCount       int     `json:"runs_count"`
	LastRunID       *string `json:"last_run_id,omitempty"`
	LastRunStatus   *string `json:"last_run_status,omitempty"`
	LastRunPhase    *string `json:"last_run_phase,omitempty"`
	LastRunStarted  *string `json:"last_run_started_at,omitempty"`
	LastRunFinished *string `json:"last_run_finished_at,omitempty"`
}

type Manager struct {
	Workspaces map[string]*Workspace
	Logger     *slog.Logger
}

func NewManager(cfg config.Config, logger *slog.Logger) (*Manager, error) {
	manager := &Manager{Workspaces: map[string]*Workspace{}, Logger: logger}
	for _, ws := range cfg.Hub.Workspaces {
		workspace := loadWorkspace(ws)
		if _, exists := manager.Workspaces[workspace.ID]; exists {
			return nil, fmt.Errorf("duplicate workspace id: %s", workspace.ID)
		}
		manager.Workspaces[workspace.ID] = workspace
	}
	return manager, nil
}

func (m *Manager) Workspace(id string) (*Workspace, error) {
	if id == "" {
		return nil, errors.New("workspace id required")
	}
	ws, ok := m.Workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	return ws, nil
}

func (m *Manager) ListSummaries() []WorkspaceSummary {
	summaries := make([]WorkspaceSummary, 0, len(m.Workspaces))
	for _, ws := range m.Workspaces {
		summaries = append(summaries, ws.Summary())
	}
	sort.Slice(summaries, func(i, j int) bool {
		return strings.ToLower(summaries[i].ID) < strings.ToLower(summaries[j].ID)
	})
	return summaries
}

func (w *Workspace) Summary() WorkspaceSummary {
	summary := WorkspaceSummary{
		ID:         w.ID,
		Name:       w.Name,
		Root:       w.Root,
		ConfigPath: w.ConfigPath,
		Status:     StatusOK,
	}
	if w.Err != nil {
		errText := w.Err.Error()
		summary.Error = &errText
		summary.Status = statusFromError(w.Err)
		return summary
	}
	if w.Store == nil {
		errText := "workspace store unavailable"
		summary.Error = &errText
		summary.Status = StatusStateError
		return summary
	}
	if runs, err := w.Store.ListRuns(); err != nil {
		errText := err.Error()
		summary.Error = &errText
		summary.Status = StatusStateError
		return summary
	} else {
		summary.RunsCount = len(runs)
		if len(runs) > 0 {
			last := runs[len(runs)-1]
			summary.LastRunID = &last.ID
			summary.LastRunStatus = &last.Status
			summary.LastRunPhase = &last.CurrentPhase
			started := last.StartedAt.UTC().Format(time.RFC3339Nano)
			summary.LastRunStarted = &started
			if last.FinishedAt != nil {
				finished := last.FinishedAt.UTC().Format(time.RFC3339Nano)
				summary.LastRunFinished = &finished
			}
		}
	}
	return summary
}

func loadWorkspace(cfg config.WorkspaceConfig) *Workspace {
	workspace := &Workspace{
		ID:         cfg.ID,
		Name:       defaultName(cfg.Name, cfg.ID),
		Root:       filepath.Clean(cfg.Root),
		ConfigPath: cfg.ConfigPath,
	}
	if workspace.ID == "" {
		workspace.Err = errors.New("workspace id required")
		return workspace
	}
	if strings.TrimSpace(workspace.Root) == "" {
		workspace.Err = errors.New("workspace root required")
		return workspace
	}
	if !filepath.IsAbs(workspace.Root) {
		absRoot, err := filepath.Abs(workspace.Root)
		if err != nil {
			workspace.Err = fmt.Errorf("resolve workspace root: %w", err)
			return workspace
		}
		workspace.Root = absRoot
	}
	configPath := cfg.ConfigPath
	if configPath == "" {
		configPath = config.DefaultConfigFile
	}
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workspace.Root, configPath)
	}
	workspace.ConfigPath = configPath

	loaded, err := config.Load(configPath)
	if err != nil {
		workspace.Err = fmt.Errorf("load config: %w", err)
		return workspace
	}
	loaded = resolveWorkspacePaths(loaded, workspace.Root)
	workspace.Config = &loaded
	workspace.Store = state.NewStore(loaded.StateDir(), loaded.RunsDir(), loaded.MemoryDir(), loaded.LogsDir(), loaded.ArtifactsDir())
	return workspace
}

func resolveWorkspacePaths(cfg config.Config, root string) config.Config {
	cfg.Paths.StateDir = resolvePath(root, cfg.Paths.StateDir)
	return cfg
}

func resolvePath(root, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}

func defaultName(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

func statusFromError(err error) string {
	if err == nil {
		return StatusOK
	}
	if errors.Is(err, os.ErrNotExist) {
		return StatusConfigMissing
	}
	if strings.Contains(err.Error(), "parse config") {
		return StatusConfigInvalid
	}
	return StatusStateError
}
