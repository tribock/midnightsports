package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tribock/midnightsports/api/internal/config"
)

// testServer baut den Server mit allen Routen, ohne Datenbank (Anfragen ohne Anmeldung erreichen sie nie).
func testServer() *Server {
	s := &Server{cfg: &config.Config{}, mux: http.NewServeMux()}
	s.routes()
	s.handler = secure(s.mux, true)
	return s
}

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

func TestRoutesRequireLogin(t *testing.T) {
	s := testServer()
	if len(s.protectedRoutes) < 40 {
		t.Fatalf("nur %d Routen gefunden", len(s.protectedRoutes))
	}
	for _, pattern := range s.protectedRoutes {
		method, path, _ := strings.Cut(pattern, " ")
		path = pathParam.ReplaceAllString(path, "0b6f8a2e-1c3d-4e5f-8a9b-0c1d2e3f4a5b")
		t.Run(pattern, func(t *testing.T) {
			req := httptest.NewRequest(method, path, strings.NewReader("{}"))
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			rec := httptest.NewRecorder()
			s.handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("ohne Anmeldung: Status %d, want 401", rec.Code)
			}
		})
	}
}

func TestCrossSiteRequestsBlocked(t *testing.T) {
	s := testServer()
	tests := []struct {
		name   string
		header map[string]string
		want   int
	}{
		{"fremde Seite (Sec-Fetch-Site)", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"fremde Seite (Origin)", map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"eigene Seite", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/events", strings.NewReader("{}"))
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			s.handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("Status %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

// Apple ruft den Callback per POST von appleid.apple.com auf, der CSRF-Schutz darf ihn nicht blockieren
func TestAppleCallbackCrossSite(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/auth/apple/callback", strings.NewReader("state=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	testServer().handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("Status %d, want 303", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/health", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options",
		"Referrer-Policy", "Permissions-Policy", "Strict-Transport-Security"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("Header %s fehlt", h)
		}
	}
}

func TestCSVText(t *testing.T) {
	tests := map[string]string{
		"Anna Muster":           "Anna Muster",
		"=HYPERLINK(\"x\")":     "'=HYPERLINK(\"x\")",
		"+41 79":                "'+41 79",
		"-1":                    "'-1",
		"@SUM(A1)":              "'@SUM(A1)",
		"":                      "",
		"Aufbau, 3 Std. (=gut)": "Aufbau, 3 Std. (=gut)",
	}
	for in, want := range tests {
		if got := csvText(in); got != want {
			t.Errorf("csvText(%q) = %q, want %q", in, got, want)
		}
	}
}
