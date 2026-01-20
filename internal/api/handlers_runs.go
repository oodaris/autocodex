package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/state"
)

// RunControlRequest describes a run control action.
type RunControlRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
	DryRun bool   `json:"dry_run"`
}

// RunControlResponse is returned when a run control request is accepted.
type RunControlResponse struct {
	RunID    string `json:"run_id"`
	Action   string `json:"action"`
	Accepted bool   `json:"accepted"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// RunControlStatus exposes the latest control state for a run.
type RunControlStatus struct {
	RunID        string     `json:"run_id"`
	Status       string     `json:"status"`
	LastAction   *string    `json:"last_action,omitempty"`
	LastActionAt *time.Time `json:"last_action_at,omitempty"`
}

// SnapshotCreateRequest configures a snapshot created via the API.
type SnapshotCreateRequest struct {
	Reason           string `json:"reason"`
	IncludeEvents    *bool  `json:"include_events,omitempty"`
	IncludeArtifacts *bool  `json:"include_artifacts,omitempty"`
	IncludeMemory    *bool  `json:"include_memory,omitempty"`
	MaxBytes         *int   `json:"max_bytes,omitempty"`
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
	case "snapshots":
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodGet:
				summaries, err := s.Store.ListSnapshots(runID)
				if err != nil {
					respondError(w, http.StatusInternalServerError, err.Error())
					s.log(r, http.StatusInternalServerError, start, err)
					return
				}
				respondJSON(w, http.StatusOK, summaries)
				s.log(r, http.StatusOK, start, nil)
				return
			case http.MethodPost:
				var req SnapshotCreateRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					respondError(w, http.StatusBadRequest, "invalid json")
					s.log(r, http.StatusBadRequest, start, err)
					return
				}
				opts := state.SnapshotOptions{
					Reason: req.Reason,
				}
				if req.MaxBytes != nil {
					opts.MaxBytes = *req.MaxBytes
				}
				if req.IncludeEvents != nil || req.IncludeArtifacts != nil || req.IncludeMemory != nil {
					includeEvents := req.IncludeEvents == nil || *req.IncludeEvents
					includeArtifacts := req.IncludeArtifacts == nil || *req.IncludeArtifacts
					includeMemory := req.IncludeMemory == nil || *req.IncludeMemory
					sources := make([]string, 0, 3)
					if includeEvents {
						sources = append(sources, "events")
					}
					if includeArtifacts {
						sources = append(sources, "artifacts")
					}
					if includeMemory {
						sources = append(sources, "memory")
					}
					if len(sources) == 0 {
						sources = []string{"none"}
					}
					opts.Sources = sources
				}
				detail, err := s.Store.CreateSnapshot(runID, opts)
				if err != nil {
					respondError(w, http.StatusNotFound, err.Error())
					s.log(r, http.StatusNotFound, start, err)
					return
				}
				respondJSON(w, http.StatusCreated, detail)
				s.log(r, http.StatusCreated, start, nil)
				return
			default:
				respondError(w, http.StatusMethodNotAllowed, "method not allowed")
				s.log(r, http.StatusMethodNotAllowed, start, nil)
				return
			}
		}
		if len(parts) == 3 {
			if r.Method != http.MethodGet {
				respondError(w, http.StatusMethodNotAllowed, "method not allowed")
				s.log(r, http.StatusMethodNotAllowed, start, nil)
				return
			}
			detail, err := s.Store.GetSnapshot(runID, parts[2])
			if err != nil {
				respondError(w, http.StatusNotFound, "snapshot not found")
				s.log(r, http.StatusNotFound, start, err)
				return
			}
			respondJSON(w, http.StatusOK, detail)
			s.log(r, http.StatusOK, start, nil)
			return
		}
		respondError(w, http.StatusNotFound, "not found")
		s.log(r, http.StatusNotFound, start, nil)
	default:
		respondError(w, http.StatusNotFound, "not found")
		s.log(r, http.StatusNotFound, start, nil)
	}
}

func validAction(action string) bool {
	switch action {
	case "resume", "stop", "cancel", "kill":
		return true
	default:
		return false
	}
}
