package api

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

func uiHandler(uiFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(uiFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		requestPath = strings.TrimSpace(requestPath)
		if requestPath == "" || strings.HasSuffix(r.URL.Path, "/") {
			requestPath = "index.html"
		} else {
			requestPath = strings.TrimPrefix(path.Clean("/"+requestPath), "/")
		}

		if info, err := fs.Stat(uiFS, requestPath); err != nil || info.IsDir() {
			requestPath = "index.html"
		}

		if requestPath == "index.html" {
			data, err := fs.ReadFile(uiFS, requestPath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(data))
			return
		}

		if strings.HasPrefix(requestPath, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		clone := r.Clone(r.Context())
		clone.URL.Path = "/" + requestPath
		fileServer.ServeHTTP(w, clone)
	})
}
