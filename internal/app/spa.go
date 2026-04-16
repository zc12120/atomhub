package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func spaHandler(distDir string) http.Handler {
	fs := http.FileServer(http.Dir(distDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/healthz" {
			http.NotFound(w, r)
			return
		}
		cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		target := filepath.Join(distDir, cleanPath)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
}
