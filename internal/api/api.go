package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

type Server struct {
	Store  *state.Store
	Logger *slog.Logger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/runs", s.handleRuns)
	mux.HandleFunc("/runs/", s.handleRunDetail)
	mux.HandleFunc("/artifacts/", s.handleArtifactDetail)
	return mux
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
