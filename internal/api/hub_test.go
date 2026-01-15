package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/state"
)

func TestHubWorkspacesEndpoint(t *testing.T) {
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

	manager, err := hub.NewManager(config.Config{
		Hub: config.HubConfig{
			Workspaces: []config.WorkspaceConfig{{
				ID:   "repo",
				Root: root,
			}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("new hub manager: %v", err)
	}

	srv := &Server{Store: newTestStore(t), Hub: manager}
	req := httptest.NewRequest(http.MethodGet, "/hub/workspaces", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}

	var summaries []hub.WorkspaceSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary")
	}
	if summaries[0].RunsCount != 1 {
		t.Fatalf("expected runs count 1, got %d", summaries[0].RunsCount)
	}
}
