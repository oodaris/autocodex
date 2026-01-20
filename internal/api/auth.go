package api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oodaris/autocodex/internal/config"
)

// AuthConfig represents runtime auth settings for the local API.
type AuthConfig struct {
	Enabled bool
	Tokens  map[string]struct{}
}

// NewAuthConfig resolves auth tokens from config and environment.
func NewAuthConfig(cfg config.AuthConfig) *AuthConfig {
	auth := &AuthConfig{Enabled: cfg.Enabled, Tokens: map[string]struct{}{}}
	if !cfg.Enabled {
		return auth
	}
	for _, token := range cfg.Tokens {
		value := strings.TrimSpace(token)
		if value == "" {
			continue
		}
		auth.Tokens[value] = struct{}{}
	}
	if cfg.TokenEnv != "" {
		if value := strings.TrimSpace(os.Getenv(cfg.TokenEnv)); value != "" {
			auth.Tokens[value] = struct{}{}
		}
	}
	return auth
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Auth == nil || !s.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		if s.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		respondError(w, http.StatusUnauthorized, "unauthorized")
		if s.Logger != nil {
			s.log(r, http.StatusUnauthorized, time.Now(), nil)
		}
	})
}

func (s *Server) authorized(r *http.Request) bool {
	if s.Auth == nil || !s.Auth.Enabled {
		return true
	}
	if token := extractToken(r); token != "" {
		_, ok := s.Auth.Tokens[token]
		return ok
	}
	return false
}

func extractToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("Authorization"); v != "" {
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return strings.TrimSpace(v[7:])
		}
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Api-Token"); v != "" {
		return strings.TrimSpace(v)
	}
	if r.URL != nil {
		if v := r.URL.Query().Get("token"); v != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
