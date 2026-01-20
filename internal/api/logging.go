package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

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
		"stage", "api",
	}
	if runID := runIDFromRequest(r); runID != "" {
		attrs = append(attrs, "run_id", runID)
	}
	if workspaceID := workspaceIDFromRequest(r); workspaceID != "" {
		attrs = append(attrs, "workspace_id", workspaceID)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	s.Logger.Info("api request", attrs...)
}

func runIDFromRequest(r *http.Request) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "runs", "artifacts":
		return parts[1]
	default:
		return ""
	}
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
