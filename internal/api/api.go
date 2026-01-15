package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/hub"
	"github.com/oodaris/autocodex/internal/state"
	"github.com/oodaris/autocodex/internal/terminal"
)

type Server struct {
	Store    *state.Store
	Logger   *slog.Logger
	Hub      *hub.Manager
	Terminal *terminal.Manager
	Auth     *AuthConfig
	Config   config.Config
	RootDir  string
}

type RunControlRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
	DryRun bool   `json:"dry_run"`
}

type RunControlResponse struct {
	RunID    string `json:"run_id"`
	Action   string `json:"action"`
	Accepted bool   `json:"accepted"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type RunControlStatus struct {
	RunID        string     `json:"run_id"`
	Status       string     `json:"status"`
	LastAction   *string    `json:"last_action,omitempty"`
	LastActionAt *time.Time `json:"last_action_at,omitempty"`
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

	handler := http.Handler(mux)
	if s.Auth != nil && s.Auth.Enabled {
		handler = s.authMiddleware(handler)
	}
	return handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	runs, err := s.Store.ListRuns()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	respondJSON(w, http.StatusOK, runs)
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := strings.TrimPrefix(r.URL.Path, "/runs/")
	if path == "" {
		respondError(w, http.StatusNotFound, "run not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}

	parts := strings.Split(path, "/")
	runID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			s.log(r, http.StatusMethodNotAllowed, start, nil)
			return
		}
		run, err := s.Store.GetRun(runID)
		if err != nil {
			respondError(w, http.StatusNotFound, "run not found")
			s.log(r, http.StatusNotFound, start, err)
			return
		}
		respondJSON(w, http.StatusOK, run)
		s.log(r, http.StatusOK, start, nil)
		return
	}

	switch parts[1] {
	case "control":
		switch r.Method {
		case http.MethodGet:
			run, err := s.Store.GetRun(runID)
			if err != nil {
				respondError(w, http.StatusNotFound, "run not found")
				s.log(r, http.StatusNotFound, start, err)
				return
			}
			control, err := s.Store.GetRunControl(runID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				s.log(r, http.StatusInternalServerError, start, err)
				return
			}
			status := RunControlStatus{
				RunID:  runID,
				Status: run.Status,
			}
			if control != nil {
				status.LastAction = control.LastAction
				status.LastActionAt = control.LastActionAt
			}
			respondJSON(w, http.StatusOK, status)
			s.log(r, http.StatusOK, start, nil)
		case http.MethodPost:
			run, err := s.Store.GetRun(runID)
			if err != nil {
				respondError(w, http.StatusNotFound, "run not found")
				s.log(r, http.StatusNotFound, start, err)
				return
			}
			var req RunControlRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondError(w, http.StatusBadRequest, "invalid json")
				s.log(r, http.StatusBadRequest, start, err)
				return
			}
			if !validAction(req.Action) {
				respondError(w, http.StatusBadRequest, "invalid action")
				s.log(r, http.StatusBadRequest, start, nil)
				return
			}
			now := time.Now().UTC()
			control := state.RunControl{
				RunID:        runID,
				Status:       run.Status,
				LastAction:   &req.Action,
				LastActionAt: &now,
				UpdatedAt:    now,
			}
			if req.Reason != "" && req.Action != "resume" {
				control.StopReason = &req.Reason
			}
			if !req.DryRun {
				if err := s.Store.SaveRunControl(control); err != nil {
					respondError(w, http.StatusInternalServerError, err.Error())
					s.log(r, http.StatusInternalServerError, start, err)
					return
				}
			}
			message := "accepted"
			if req.DryRun {
				message = "dry run"
			}
			respondJSON(w, http.StatusAccepted, RunControlResponse{
				RunID:    runID,
				Action:   req.Action,
				Accepted: true,
				Status:   run.Status,
				Message:  message,
			})
			s.log(r, http.StatusAccepted, start, nil)
		default:
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			s.log(r, http.StatusMethodNotAllowed, start, nil)
		}
	case "events":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			s.log(r, http.StatusMethodNotAllowed, start, nil)
			return
		}
		events, err := s.Store.ListEvents(runID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		respondJSON(w, http.StatusOK, events)
		s.log(r, http.StatusOK, start, nil)
	case "artifacts":
		if r.Method != http.MethodGet {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			s.log(r, http.StatusMethodNotAllowed, start, nil)
			return
		}
		artifacts, err := s.Store.ListArtifacts(runID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			s.log(r, http.StatusInternalServerError, start, err)
			return
		}
		respondJSON(w, http.StatusOK, artifacts)
		s.log(r, http.StatusOK, start, nil)
	default:
		respondError(w, http.StatusNotFound, "not found")
		s.log(r, http.StatusNotFound, start, nil)
	}
}

func (s *Server) handleArtifactDetail(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/artifacts/")
	if id == "" {
		respondError(w, http.StatusNotFound, "artifact not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	artifact, err := s.Store.GetArtifact(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "artifact not found")
		s.log(r, http.StatusNotFound, start, err)
		return
	}
	respondJSON(w, http.StatusOK, artifact)
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) handleMemoryDocs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	docs, err := s.Store.ListMemoryDocs()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	respondJSON(w, http.StatusOK, docs)
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) handleMemoryDocDetail(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/memory/")
	if name == "" {
		respondError(w, http.StatusNotFound, "memory doc not found")
		s.log(r, http.StatusNotFound, start, nil)
		return
	}
	doc, err := s.Store.GetMemoryDoc(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			respondError(w, http.StatusNotFound, "memory doc not found")
			s.log(r, http.StatusNotFound, start, err)
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	respondJSON(w, http.StatusOK, doc)
	s.log(r, http.StatusOK, start, nil)
}

func (s *Server) log(r *http.Request, status int, start time.Time, err error) {
	if s.Logger == nil {
		return
	}
	latency := time.Since(start).Milliseconds()
	traceID := traceIDFromRequest(r)
	tenantID := tenantIDFromRequest(r)
	attrs := []any{
		"trace_id", traceID,
		"tenant_id", tenantID,
		"route", r.URL.Path,
		"status", status,
		"latency_ms", latency,
	}
	if workspaceID := workspaceIDFromRequest(r); workspaceID != "" {
		attrs = append(attrs, "workspace_id", workspaceID)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	s.Logger.Info("api request", attrs...)
}

func respondJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func validAction(action string) bool {
	switch action {
	case "resume", "stop", "cancel", "kill":
		return true
	default:
		return false
	}
}

func traceIDFromRequest(r *http.Request) string {
	if r == nil {
		return "trace-unknown"
	}
	if v := r.Header.Get("X-Trace-Id"); v != "" {
		return v
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "trace-unknown"
	}
	return "trace-" + hex.EncodeToString(buf)
}

func tenantIDFromRequest(r *http.Request) string {
	if r == nil {
		return "local"
	}
	if v := r.Header.Get("X-Tenant-Id"); v != "" {
		return v
	}
	if v := os.Getenv("AUTOCODEX_TENANT_ID"); v != "" {
		return v
	}
	return "local"
}

func workspaceIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Workspace-Id"); v != "" {
		return v
	}
	return ""
}
