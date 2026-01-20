package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oodaris/autocodex/internal/config"
	"github.com/oodaris/autocodex/internal/state"
)

func newTestServer(t *testing.T) *Server {
	store := newTestStore(t)
	return &Server{Store: store, Config: config.Config{}}
}

func TestHandleHealth(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected status ok")
	}
}

func TestHandleRunsEmpty(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var runs []state.Run
	if err := json.Unmarshal(res.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected empty runs")
	}
}

func TestHandleRunsMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader(`{}`))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.Code)
	}
}

func TestHandleRunsWithRun(t *testing.T) {
	srv := newTestServer(t)
	if _, err := srv.Store.CreateRun(); err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var runs []state.Run
	if err := json.Unmarshal(res.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
}

func TestHandleRunDetailMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID, strings.NewReader(`{}`))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.Code)
	}
}

func TestHandleRunDetailNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/runs/does-not-exist", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestHandleRunEventsEmpty(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/events", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var events []state.RunEvent
	if err := json.Unmarshal(res.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events")
	}
}

func TestHandleRunControlDryRun(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	body := `{"action":"stop","dry_run":true}`
	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/control", strings.NewReader(body))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}
}

func TestHandleRunControlInvalidAction(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	body := `{"action":"nope"}`
	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/control", strings.NewReader(body))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestHandleRunControlInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/control", strings.NewReader("not-json"))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestHandleSnapshotsInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/snapshots", strings.NewReader("nope"))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestHandleSnapshotNotFound(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/snapshots/missing", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestHandleMemoryDocsAndDetail(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.Store.AppendMemoryDoc("TODO.md", "- item"); err != nil {
		t.Fatalf("append memory: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/memory", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/memory/TODO.md", nil)
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestHandleMemoryDocsMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/memory", strings.NewReader(`{}`))
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.Code)
	}
}

func TestHandleMemoryDocNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/memory/NOPE.md", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestHandleArtifactDetail(t *testing.T) {
	srv := newTestServer(t)
	run, err := srv.Store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	artifactDir := filepath.Join(srv.Store.RunsDir, run.ID, "artifacts")
	path := filepath.Join(artifactDir, "test.txt")
	if err := os.WriteFile(path, []byte("artifact"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	// Ensure modtime differs to avoid flakiness in some filesystems.
	time.Sleep(5 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/artifacts/"+run.ID+":test.txt", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestHandleArtifactDetailNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/artifacts/invalid", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestAuthMiddlewareRequiresToken(t *testing.T) {
	srv := newTestServer(t)
	srv.Auth = &AuthConfig{Enabled: true, Tokens: map[string]struct{}{"token-1": {}}}
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/runs", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	res = httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}
