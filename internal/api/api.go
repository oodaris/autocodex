package api

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/state"
	"github.com/oodaris/autocodex/internal/terminal"
)

// Server wires storage, auth, and optional UI hosting into an HTTP handler.
type Server struct {
	Store    *state.Store
	Logger   *slog.Logger
	Hub      *hub.Manager
	Terminal *terminal.Manager
	Auth     *AuthConfig
	Config   config.Config
	RootDir  string
	UIFS     fs.FS
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/runs", s.handleRuns)
	mux.HandleFunc("/runs/", s.handleRunDetail)
	mux.HandleFunc("/artifacts/", s.handleArtifactDetail)
	mux.HandleFunc("/memory", s.handleMemoryDocs)
	mux.HandleFunc("/memory/", s.handleMemoryDocDetail)
	mux.HandleFunc("/hub/workspaces", s.handleHubWorkspaces)
	mux.HandleFunc("/hub/workspaces/", s.handleHubWorkspace)
	mux.HandleFunc("/terminal/sessions", s.handleTerminalSessions)
	mux.HandleFunc("/terminal/sessions/", s.handleTerminalSession)
	if s.UIFS != nil {
		mux.Handle("/", uiHandler(s.UIFS))
	}

	handler := http.Handler(mux)
	if s.Auth != nil && s.Auth.Enabled {
		handler = s.authMiddleware(handler)
	}
	return s.corsMiddleware(handler)
}
