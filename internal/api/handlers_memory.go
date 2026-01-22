package api

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s *Server) handleMemoryDocs(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		s.log(r, http.StatusMethodNotAllowed, start, nil)
		return
	}
	docs, err := s.Store.ListMemoryDocs()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	if err := respondJSON(w, http.StatusOK, docs); err != nil {
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
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
		respondError(w, http.StatusInternalServerError, "internal error")
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	if err := respondJSON(w, http.StatusOK, doc); err != nil {
		s.log(r, http.StatusInternalServerError, start, err)
		return
	}
	s.log(r, http.StatusOK, start, nil)
}
