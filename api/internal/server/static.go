package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaHandler liefert das gebaute Frontend aus. Unbekannte Pfade erhalten index.html,
// damit das Routing im Browser funktioniert.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}
