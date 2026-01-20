package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
)

func TestCorsMiddlewareOptions(t *testing.T) {
	srv := &Server{
		Config: config.Config{
			UI: config.UIConfig{Origin: "http://example.com"},
		},
	}
	handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/runs", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Fatalf("expected allowed origin header")
	}
}

func TestCorsMiddlewarePassThrough(t *testing.T) {
	called := false
	srv := &Server{
		Config: config.Config{
			UI: config.UIConfig{Origin: "http://example.com"},
		},
	}
	handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Fatalf("expected allowed origin header")
	}
}
