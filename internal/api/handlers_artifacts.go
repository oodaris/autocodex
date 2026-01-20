package api

import (
	"net/http"
	"strings"
	"time"
)

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
