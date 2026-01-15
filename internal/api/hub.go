package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/hub"
)

func (s *Server) handleHubWorkspaces(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	if s.Hub == nil {
		respondError(w, http.StatusNotFound, "hub not enabled")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	summaries := s.Hub.ListSummaries()
	respondJSON(w, http.StatusOK, summaries)
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) handleHubWorkspace(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if s.Hub == nil {
		respondError(w, http.StatusNotFound, "hub not enabled")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/hub/workspaces/")
	if path == "" {
		respondError(w, http.StatusNotFound, "workspace not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	parts := strings.SplitN(path, "/", 2)
	workspaceID := parts[0]
	ws, err := s.Hub.Workspace(workspaceID)
	if err != nil {
		respondError(w, http.StatusNotFound, "workspace not found")
		s.log(r, http.StatusNotFound, start, err)
		return
	}
	r.Header.Set("X-Workspace-Id", workspaceID)
	if ws.Err != nil {
		summary := ws.Summary()
		respondJSON(w, http.StatusServiceUnavailable, summary)
		s.log(r, http.StatusServiceUnavailable, start, ws.Err)
		return
	}

	if len(parts) == 1 || parts[1] == "" {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			s.log(r, http.StatusMethodNotAllowed, start, nil)
			return
		}
		summary := ws.Summary()
		respondJSON(w, http.StatusOK, summary)
		s.log(r, http.StatusOK, start, nil)
		return
	}

	remainder := parts[1]
	proxy := s.workspaceServer(ws)

	cloned := r.Clone(r.Context())
	cloned.Header.Set("X-Workspace-Id", workspaceID)

	switch {
	case remainder == "runs" || remainder == "runs/":
		cloned.URL.Path = "/runs"
		proxy.handleRuns(w, cloned)
	case strings.HasPrefix(remainder, "runs/"):
		cloned.URL.Path = "/runs/" + strings.TrimPrefix(remainder, "runs/")
		proxy.handleRunDetail(w, cloned)
	case strings.HasPrefix(remainder, "artifacts/"):
		cloned.URL.Path = "/artifacts/" + strings.TrimPrefix(remainder, "artifacts/")
		proxy.handleArtifactDetail(w, cloned)
	case remainder == "memory" || remainder == "memory/":
		cloned.URL.Path = "/memory"
		proxy.handleMemoryDocs(w, cloned)
	case strings.HasPrefix(remainder, "memory/"):
		cloned.URL.Path = "/memory/" + strings.TrimPrefix(remainder, "memory/")
		proxy.handleMemoryDocDetail(w, cloned)
	default:
		respondError(w, http.StatusNotFound, "not found")
		s.log(r, http.StatusNotFound, start, nil)
	}
}

func (s *Server) workspaceServer(ws *hub.Workspace) *Server {
	return &Server{
		Store:  ws.Store,
		Logger: s.Logger,
	}
}
