package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTraceIDFromRequest(t *testing.T) {
	if traceIDFromRequest(nil) != "trace-unknown" {
		t.Fatalf("expected trace-unknown for nil request")
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Trace-Id", "trace-abc")
	if traceIDFromRequest(req) != "trace-abc" {
		t.Fatalf("expected header trace id")
	}
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	trace := traceIDFromRequest(req)
	if !strings.HasPrefix(trace, "trace-") {
		t.Fatalf("expected trace prefix, got %s", trace)
	}
}

func TestTenantIDFromRequest(t *testing.T) {
	if tenantIDFromRequest(nil) != "local" {
		t.Fatalf("expected local for nil request")
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Tenant-Id", "tenant-1")
	if tenantIDFromRequest(req) != "tenant-1" {
		t.Fatalf("expected header tenant id")
	}
	t.Setenv("AUTOCODEX_TENANT_ID", "tenant-env")
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	if tenantIDFromRequest(req) != "tenant-env" {
		t.Fatalf("expected env tenant id")
	}
}

func TestWorkspaceIDFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Workspace-Id", "ws-1")
	if workspaceIDFromRequest(req) != "ws-1" {
		t.Fatalf("expected workspace id")
	}
	if workspaceIDFromRequest(nil) != "" {
		t.Fatalf("expected empty workspace for nil request")
	}
}
