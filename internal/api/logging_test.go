package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type logRecord struct {
	msg   string
	attrs map[string]any
}

type captureStore struct {
	mu      sync.Mutex
	records []logRecord
}

type captureHandler struct {
	store *captureStore
	base  []slog.Attr
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{}
	for _, attr := range h.base {
		attrs[attr.Key] = attr.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.store.mu.Lock()
	h.store.records = append(h.store.records, logRecord{msg: r.Message, attrs: attrs})
	h.store.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Attr, 0, len(h.base)+len(attrs))
	next = append(next, h.base...)
	next = append(next, attrs...)
	return &captureHandler{store: h.store, base: next}
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	_ = name
	return h
}

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

func TestAPILogIncludesStageAndRunID(t *testing.T) {
	store := &captureStore{}
	logger := slog.New(&captureHandler{store: store})
	server := &Server{Logger: logger}

	req := httptest.NewRequest(http.MethodGet, "/runs/run-123", nil)
	start := time.Now()
	server.log(req, http.StatusOK, start, nil)

	if len(store.records) == 0 {
		t.Fatalf("expected log record")
	}
	record := store.records[len(store.records)-1]
	if got := record.attrs["stage"]; got != "api" {
		t.Fatalf("expected stage=api, got %v", got)
	}
	if got := record.attrs["run_id"]; got != "run-123" {
		t.Fatalf("expected run_id run-123, got %v", got)
	}
}
