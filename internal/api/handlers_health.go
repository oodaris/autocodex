package api

import (
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if err := respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	s.log(r, http.StatusOK, start, nil)
}
