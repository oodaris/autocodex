package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
)

func traceIDFromRequest(r *http.Request) string {
	if r == nil {
		return "trace-unknown"
	}
	if v := r.Header.Get("X-Trace-Id"); v != "" {
		return v
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "trace-unknown"
	}
	return "trace-" + hex.EncodeToString(buf)
}

func tenantIDFromRequest(r *http.Request) string {
	if r == nil {
		return "local"
	}
	if v := r.Header.Get("X-Tenant-Id"); v != "" {
		return v
	}
	if v := os.Getenv("AUTOCODEX_TENANT_ID"); v != "" {
		return v
	}
	return "local"
}

func workspaceIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Workspace-Id"); v != "" {
		return v
	}
	return ""
}
