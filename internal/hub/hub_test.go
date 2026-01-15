package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func TestHubWorkspaceSummary(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "autocodex.yaml")
	if err := os.WriteFile(configPath, []byte("version: v1\nmode: yolo\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Paths.StateDir = filepath.Join(root, cfg.Paths.StateDir)
	store := state.NewStore(cfg.StateDir(), cfg.RunsDir(), cfg.MemoryDir(), cfg.LogsDir(), cfg.ArtifactsDir())
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	if _, err := store.CreateRun(); err != nil {
		t.Fatalf("create run: %v", err)
	}

	manager, err := NewManager(config.Config{
		Hub: config.HubConfig{
			Workspaces: []config.WorkspaceConfig{{
				ID:   "demo",
				Name: "Demo",
				Root: root,
			}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	summaries := manager.ListSummaries()
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary")
	}
	if summaries[0].Status != StatusOK {
		t.Fatalf("expected ok status, got %s", summaries[0].Status)
	}
	if summaries[0].RunsCount != 1 {
		t.Fatalf("expected 1 run")
	}
}

func TestHubWorkspaceMissingConfig(t *testing.T) {
	root := t.TempDir()
	manager, err := NewManager(config.Config{
		Hub: config.HubConfig{
			Workspaces: []config.WorkspaceConfig{{
				ID:   "missing",
				Root: root,
			}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if len(manager.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace")
	}
	summary := manager.ListSummaries()[0]
	if summary.Status != StatusConfigMissing {
		t.Fatalf("expected missing config status, got %s", summary.Status)
	}
	if summary.Error == nil {
		t.Fatalf("expected error message")
	}
}
