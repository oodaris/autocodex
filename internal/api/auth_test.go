package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/oodaris/autocodex/internal/config"
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

func TestExtractToken(t *testing.T) {
	if extractToken(nil) != "" {
		t.Fatalf("expected empty token for nil request")
	}
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	req.Header.Set("Authorization", "Bearer token-abc")
	if extractToken(req) != "token-abc" {
		t.Fatalf("expected bearer token")
	}
	req = httptest.NewRequest(http.MethodGet, "/runs", nil)
	req.Header.Set("Authorization", "token-raw")
	if extractToken(req) != "token-raw" {
		t.Fatalf("expected raw token")
	}
	req = httptest.NewRequest(http.MethodGet, "/runs", nil)
	req.Header.Set("X-Api-Token", "token-x")
	if extractToken(req) != "token-x" {
		t.Fatalf("expected x-api-token")
	}
	req = httptest.NewRequest(http.MethodGet, "/runs?token=token-q", nil)
	if extractToken(req) != "token-q" {
		t.Fatalf("expected query token")
	}
}

func TestNewAuthConfigDisabled(t *testing.T) {
	auth := NewAuthConfig(config.AuthConfig{Enabled: false, Tokens: []string{"token"}})
	if auth.Enabled {
		t.Fatalf("expected auth disabled")
	}
	if len(auth.Tokens) != 0 {
		t.Fatalf("expected no tokens when disabled")
	}
}

func TestNewAuthConfigTokensAndEnv(t *testing.T) {
	os.Setenv("AUTOCODEX_TEST_TOKEN", "env-token")
	defer os.Unsetenv("AUTOCODEX_TEST_TOKEN")

	auth := NewAuthConfig(config.AuthConfig{
		Enabled:  true,
		TokenEnv: "AUTOCODEX_TEST_TOKEN",
		Tokens:   []string{" token-a ", "", "token-b"},
	})
	if !auth.Enabled {
		t.Fatalf("expected auth enabled")
	}
	for _, token := range []string{"token-a", "token-b", "env-token"} {
		if _, ok := auth.Tokens[token]; !ok {
			t.Fatalf("expected token %s in auth config", token)
		}
	}
}

func TestAuthorizedHelper(t *testing.T) {
	srv := &Server{
		Auth: &AuthConfig{
			Enabled: true,
			Tokens:  map[string]struct{}{"token-xyz": {}},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	if srv.authorized(req) {
		t.Fatalf("expected unauthorized without token")
	}
	req.Header.Set("X-Api-Token", "token-xyz")
	if !srv.authorized(req) {
		t.Fatalf("expected authorized with token")
	}
}
