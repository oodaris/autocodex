package api

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestUIHandlerFallback(t *testing.T) {
	uiFS := fstest.MapFS{
		"index.html":       &fstest.MapFile{Data: []byte("<html>app</html>")},
		"assets/app.js":    &fstest.MapFile{Data: []byte("console.log('ok')")},
		"assets/style.css": &fstest.MapFile{Data: []byte("body{}")},
	}
	handler := uiHandler(fs.FS(uiFS))

	tests := []struct {
		path     string
		wantBody string
	}{
		{path: "/", wantBody: "<html>app</html>"},
		{path: "/unknown", wantBody: "<html>app</html>"},
		{path: "/assets/app.js", wantBody: "console.log('ok')"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		body, _ := io.ReadAll(rec.Body)
		if !strings.Contains(string(body), tt.wantBody) {
			t.Fatalf("path %s status %d response %q missing %q", tt.path, rec.Code, string(body), tt.wantBody)
		}
	}
}
