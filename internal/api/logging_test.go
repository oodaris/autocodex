package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	respondJSON(rr, http.StatusCreated, map[string]string{"ok": "yes"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content-type, got %s", ct)
	}
	if !strings.Contains(rr.Body.String(), `"ok":"yes"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestRespondError(t *testing.T) {
	rr := httptest.NewRecorder()
	respondError(rr, http.StatusBadRequest, "bad input")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"error":"bad input"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
