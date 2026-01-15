package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	store := newTestStore(t)
	srv := &Server{
		Store: store,
		Auth: &AuthConfig{
			Enabled: true,
			Tokens:  map[string]struct{}{"token-123": {}},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/runs", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
