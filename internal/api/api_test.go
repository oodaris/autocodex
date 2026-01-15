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

	"github.com/oodaris/autocodex/internal/state"
)

func TestRunsEndpoint(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	srv := &Server{Store: store}
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}

	var runs []state.Run
	if err := json.Unmarshal(rr.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(runs) == 0 || runs[0].ID != run.ID {
		t.Fatalf("unexpected runs list")
	}
}

func TestArtifactsEndpoint(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	artifactPath := filepath.Join(store.RunsDir, run.ID, "artifacts", "phase.txt")
	if err := os.WriteFile(artifactPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	srv := &Server{Store: store}
	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/artifacts", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}

	var artifacts []state.Artifact
	if err := json.Unmarshal(rr.Body.Bytes(), &artifacts); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact")
	}
	if artifacts[0].RunID != run.ID {
		t.Fatalf("unexpected run id")
	}
}

func TestMemoryDocsEndpoints(t *testing.T) {
	store := newTestStore(t)

	srv := &Server{Store: store}

	req := httptest.NewRequest(http.MethodGet, "/memory", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}

	var docs []state.MemoryDocSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &docs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(docs) == 0 {
		t.Fatalf("expected memory docs")
	}

	req = httptest.NewRequest(http.MethodGet, "/memory/"+docs[0].Name, nil)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("detail status: %d", rr.Code)
	}

	var detail state.MemoryDocDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.Name != docs[0].Name {
		t.Fatalf("unexpected doc name")
	}
	if detail.Content == "" {
		t.Fatalf("expected content")
	}
}

func TestRunControlEndpoints(t *testing.T) {
	store := newTestStore(t)
	run, err := store.CreateRun()
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	srv := &Server{Store: store}

	req := httptest.NewRequest(http.MethodGet, "/runs/"+run.ID+"/control", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}

	var status RunControlStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.RunID != run.ID {
		t.Fatalf("unexpected run id")
	}

	body := strings.NewReader(`{"action":"stop","reason":"pause"}`)
	req = httptest.NewRequest(http.MethodPost, "/runs/"+run.ID+"/control", body)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status: %d", rr.Code)
	}

	control, err := store.GetRunControl(run.ID)
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if control == nil || control.LastAction == nil || *control.LastAction != "stop" {
		t.Fatalf("expected stop action saved")
	}
	if control.StopReason == nil || *control.StopReason != "pause" {
		t.Fatalf("expected stop reason saved")
	}
}

func newTestStore(t *testing.T) *state.Store {
	base := t.TempDir()
	store := state.NewStore(
		filepath.Join(base, "state"),
		filepath.Join(base, "runs"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "artifacts"),
	)
	if err := store.InitDirs(); err != nil {
		t.Fatalf("init dirs: %v", err)
	}
	if err := store.EnsureMemoryDocs(); err != nil {
		t.Fatalf("memory docs: %v", err)
	}

	// ensure modtime differs in tests that inspect it
	time.Sleep(5 * time.Millisecond)
	return store
}
