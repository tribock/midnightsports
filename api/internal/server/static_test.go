package server

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSPAHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644)
	os.Mkdir(filepath.Join(dir, "assets"), 0o755)
	script := strings.Repeat("console.log('midnightsports');", 100)
	os.WriteFile(filepath.Join(dir, "assets", "index-abc.js"), []byte(script), 0o644)
	h := spaHandler(dir)

	get := func(path, encoding string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if encoding != "" {
			req.Header.Set("Accept-Encoding", encoding)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	t.Run("Assets komprimiert", func(t *testing.T) {
		rec := get("/assets/index-abc.js", "gzip, br")
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatal("nicht komprimiert")
		}
		if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
			t.Error("Cache-Control fehlt")
		}
		zr, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		if body, _ := io.ReadAll(zr); string(body) != script {
			t.Error("Inhalt nach dem Entpacken verschieden")
		}
	})

	t.Run("ohne gzip unkomprimiert", func(t *testing.T) {
		rec := get("/assets/index-abc.js", "")
		if rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != script {
			t.Error("Browser ohne gzip erhält komprimierte Daten")
		}
	})

	t.Run("unbekannter Pfad liefert die App", func(t *testing.T) {
		rec := get("/events/12", "gzip")
		if rec.Body.String() != "<html>app</html>" || rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("got %q, Cache-Control %q", rec.Body.String(), rec.Header().Get("Cache-Control"))
		}
	})
}
