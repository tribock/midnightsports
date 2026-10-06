// Package server enthält die HTTP-API.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tribock/midnightsports/api/internal/config"
	"github.com/tribock/midnightsports/api/internal/wallet"
)

const (
	RoleAdmin         = "admin"
	RoleProjektleiter = "projektleiter"
	RoleTeilnehmer    = "teilnehmer"
)

var (
	anyRole = []string(nil)
	leaders = []string{RoleProjektleiter, RoleAdmin}
	admins  = []string{RoleAdmin}
)

// BuildInfo beschreibt die laufende Version der App.
type BuildInfo struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

type Server struct {
	build        BuildInfo
	cfg          *config.Config
	db           *pgxpool.Pool
	oauth        *oauth2.Config
	tz           *time.Location
	mux          *http.ServeMux
	appleWallet  *wallet.Apple
	googleWallet *wallet.Google
}

func New(cfg *config.Config, db *pgxpool.Pool, build BuildInfo) (*Server, error) {
	tz, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return nil, err
	}
	s := &Server{
		build: build,
		cfg:   cfg,
		db:    db,
		tz:    tz,
		mux:   http.NewServeMux(),
		oauth: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.OAuthRedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
	if s.appleWallet, err = wallet.NewApple(cfg.ApplePassTypeID, cfg.AppleTeamID, cfg.ApplePassPEMFile); err != nil {
		return nil, err
	}
	if s.googleWallet, err = wallet.NewGoogle(cfg.GoogleWalletIssuerID, cfg.GoogleWalletKeyFile, cfg.BaseURL()); err != nil {
		return nil, err
	}
	s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	s.mux.ServeHTTP(rec, r)
	slog.Debug("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
}

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.build)
	})
	m.HandleFunc("GET /api/config", s.getConfig)
	m.HandleFunc("GET /api/auth/login", s.login)
	m.HandleFunc("GET /api/auth/callback", s.callback)
	m.HandleFunc("POST /api/auth/dev-login", s.devLogin)
	m.HandleFunc("POST /api/auth/logout", s.logout)

	s.route("GET /api/me", anyRole, s.getMe)
	s.route("PUT /api/me", anyRole, s.updateMe)
	s.route("DELETE /api/me", anyRole, s.deleteMe)
	s.route("GET /api/me/qr", anyRole, s.getMyQR)
	s.route("GET /api/me/wallet/apple", anyRole, s.getAppleWalletPass)
	s.route("GET /api/me/wallet/google", anyRole, s.getGoogleWalletLink)
	m.HandleFunc("GET /api/wallet/logo.png", s.getWalletLogo)

	s.route("GET /api/events", anyRole, s.listEvents)
	s.route("POST /api/events", leaders, s.createEvent)
	s.route("GET /api/events/{id}", anyRole, s.getEvent)
	s.route("PUT /api/events/{id}", leaders, s.updateEvent)
	s.route("POST /api/events/{id}/cancel", leaders, s.cancelEvent)
	s.route("POST /api/events/{id}/registration", anyRole, s.register)
	s.route("DELETE /api/events/{id}/registration", anyRole, s.unregister)
	s.route("GET /api/events/{id}/participants", leaders, s.listParticipants)
	s.route("POST /api/events/{id}/checkins", leaders, s.checkIn)
	s.route("DELETE /api/events/{id}/checkins/{checkinID}", leaders, s.deleteCheckIn)
	s.route("GET /api/events/{id}/report", leaders, s.getReport)
	s.route("PUT /api/events/{id}/report", leaders, s.saveReport)

	s.route("GET /api/users", leaders, s.listUsers)
	s.route("GET /api/users/{id}", leaders, s.getUser)
	s.route("PUT /api/users/{id}/role", admins, s.setRole)

	s.route("GET /api/locations", anyRole, s.listLocations)
	s.route("POST /api/locations", admins, s.createLocation)
	s.route("PUT /api/locations/{id}", admins, s.updateLocation)

	s.route("GET /api/time-entries", leaders, s.listTimeEntries)
	s.route("POST /api/time-entries", leaders, s.createTimeEntry)
	s.route("DELETE /api/time-entries/{id}", leaders, s.deleteTimeEntry)
	s.route("GET /api/time-entries/export.csv", admins, s.exportTimeEntries)

	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, errNotFound)
	})
	if s.cfg.StaticDir != "" {
		m.Handle("/", spaHandler(s.cfg.StaticDir))
	}
}

type handlerFunc func(w http.ResponseWriter, r *http.Request, u *User) error

// route registriert einen Handler, der einen angemeldeten Benutzer mit einer der Rollen verlangt.
func (s *Server) route(pattern string, roles []string, h handlerFunc) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		u, err := s.sessionUser(r)
		if err != nil {
			writeError(w, err)
			return
		}
		if roles != nil && !slices.Contains(roles, u.Role) {
			writeError(w, errForbidden)
			return
		}
		if err := h(w, r, u); err != nil {
			writeError(w, err)
		}
	})
}

type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }

func errorf(status int, msg string) error { return &apiError{status, msg} }

var (
	errUnauthorized = errorf(http.StatusUnauthorized, "Nicht angemeldet")
	errForbidden    = errorf(http.StatusForbidden, "Keine Berechtigung")
	errNotFound     = errorf(http.StatusNotFound, "Nicht gefunden")
	errIncomplete   = errorf(http.StatusConflict, "Bitte zuerst das Profil vollständig ausfüllen")
)

func writeError(w http.ResponseWriter, err error) {
	var ae *apiError
	switch {
	case errors.As(err, &ae):
	case errors.Is(err, pgx.ErrNoRows):
		ae = errNotFound.(*apiError)
	default:
		slog.Error("request failed", "error", err)
		ae = &apiError{http.StatusInternalServerError, "Interner Fehler"}
	}
	writeJSON(w, ae.status, map[string]string{"error": ae.msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return errorf(http.StatusBadRequest, "Ungültige Anfrage")
	}
	return nil
}

func pathInt(r *http.Request, name string) (int, error) {
	n, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		return 0, errNotFound
	}
	return n, nil
}

func isLeader(u *User) bool { return slices.Contains(leaders, u.Role) }

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
