package server

import (
	"bytes"
	"compress/gzip"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// spaHandler liefert das gebaute Frontend aus. Unbekannte Pfade erhalten index.html,
// damit das Routing im Browser funktioniert.
func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	gz := &gzipCache{files: map[string][]byte{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			files.ServeHTTP(w, r)
			return
		}
		// Dateien in /assets/ tragen einen Hash im Namen und ändern sich nie
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Add("Vary", "Accept-Encoding")
		if compressible(path) && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			if data, err := gz.get(path); err == nil {
				w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
				w.Header().Set("Content-Encoding", "gzip")
				w.Write(data)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}

// compressible: Textdateien lohnen sich, Bilder und Schriften sind bereits komprimiert.
func compressible(path string) bool {
	switch filepath.Ext(path) {
	case ".js", ".css", ".svg", ".json", ".html":
		return true
	}
	return false
}

// gzipCache komprimiert jede Datei nur einmal und behält sie im Speicher.
// Nur für unveränderliche Dateien gedacht (/assets/), das gebaute Frontend ist wenige hundert KB gross.
type gzipCache struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (c *gzipCache) get(path string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if data, ok := c.files[path]; ok {
		return data, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	zw.Write(raw)
	if err := zw.Close(); err != nil {
		return nil, err
	}
	c.files[path] = buf.Bytes()
	return c.files[path], nil
}
