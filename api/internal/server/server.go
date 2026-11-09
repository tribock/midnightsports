// Package server enthält die HTTP-API.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
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
	RoleMitarbeiter   = "mitarbeiter"
	RoleTeilnehmer    = "teilnehmer"
)

// Rollen in einem Projekt (project_members.role). Admin ist der Projekt-Admin.
var (
	allRoles = []string{RoleAdmin, RoleProjektleiter, RoleMitarbeiter, RoleTeilnehmer}
	anyRole  = []string(nil)
	// Teilnehmer erfassen und einchecken
	leaders = []string{RoleProjektleiter, RoleMitarbeiter, RoleAdmin}
	// Events erfassen und bearbeiten, Berichte, Auszahlung und Zeiterfassung (ohne Mitarbeiter)
	projectLeads = []string{RoleProjektleiter, RoleAdmin}
	admins       = []string{RoleAdmin}
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
	apple        *appleLogin // nil, wenn der Login mit Apple nicht konfiguriert ist
	tz           *time.Location
	mux          *http.ServeMux
	handler      http.Handler // mux mit Sicherheits-Headern und CSRF-Schutz
	appleWallet  *wallet.Apple
	googleWallet *wallet.Google
	// Muster aller Routen mit Anmeldung (für Tests)
	protectedRoutes []string
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
	if s.apple, err = newAppleLogin(cfg); err != nil {
		return nil, err
	}
	if s.appleWallet, err = wallet.NewApple(cfg.ApplePassTypeID, cfg.AppleTeamID, cfg.ApplePassPEMFile); err != nil {
		return nil, err
	}
	if s.googleWallet, err = wallet.NewGoogle(cfg.GoogleWalletIssuerID, cfg.GoogleWalletKeyFile, cfg.BaseURL()); err != nil {
		return nil, err
	}
	s.routes()
	s.handler = secure(s.mux, cfg.SecureCookies())
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	s.handler.ServeHTTP(rec, r)
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
	m.HandleFunc("GET /api/auth/apple/login", s.appleLoginStart)
	m.HandleFunc("POST /api/auth/apple/callback", s.appleCallback)
	m.HandleFunc("POST /api/auth/dev-login", s.devLogin)
	m.HandleFunc("POST /api/auth/logout", s.logout)
	m.HandleFunc("GET /api/wallet/logo.png", s.getWalletLogo)
	// Öffentlich für die Login-Seite über den Projekt-Link: nur Name, Region und Logo aktiver Projekte
	m.HandleFunc("GET /api/public/projects/{project}", s.publicProject)
	m.HandleFunc("GET /api/public/projects/{project}/logo", s.projectLogo)

	// Person (global)
	s.route("GET /api/me", s.getMe)
	s.route("PUT /api/me", s.updateMe)
	s.route("DELETE /api/me", s.deleteMe)
	s.route("GET /api/me/qr", s.getMyQR)
	s.route("GET /api/me/wallet/apple", s.getAppleWalletPass)
	s.route("GET /api/me/wallet/google", s.getGoogleWalletLink)

	// Projekte suchen und sich anmelden
	s.route("GET /api/projects", s.searchProjects)
	s.route("POST /api/projects/{project}/join", s.joinProject)

	// Globale Administration
	s.adminRoute("GET /api/admin/projects", s.listAdminProjects)
	s.adminRoute("POST /api/admin/projects", s.createProject)
	s.adminRoute("PUT /api/admin/projects/{project}", s.updateProject)
	s.adminRoute("DELETE /api/admin/projects/{project}/shares/{partner}", s.removeShare)
	s.adminRoute("GET /api/admin/users", s.listAdminUsers)
	s.adminRoute("GET /api/admin/users/{id}", s.getAdminUser)
	s.adminRoute("PUT /api/admin/users/{id}", s.updateAdminUser)
	s.adminRoute("DELETE /api/admin/users/{id}", s.deleteAdminUser)
	s.adminRoute("GET /api/admin/audit", s.listAudit)

	// Projekt: alle Routen unter /api/p/{project}/, Rollen gelten pro Projekt
	p := func(pattern string, roles []string, h memberHandler) {
		method, path, _ := strings.Cut(pattern, " ")
		s.projectRoute(method+" /api/p/{project}"+path, roles, h)
	}
	p("GET /me", anyRole, s.getProjectMe)

	p("GET /events", anyRole, s.listEvents)
	// Events erfassen, bearbeiten und absagen nur Projektleiter und Admins, Mitarbeiter nur einchecken
	p("POST /events", projectLeads, s.createEvent)
	p("GET /events/{id}", anyRole, s.getEvent)
	p("PUT /events/{id}", projectLeads, s.updateEvent)
	p("POST /events/{id}/cancel", projectLeads, s.cancelEvent)
	p("POST /events/{id}/registration", anyRole, s.register)
	p("DELETE /events/{id}/registration", anyRole, s.unregister)
	p("GET /events/{id}/participants", leaders, s.listParticipants)
	p("POST /events/{id}/checkins", leaders, s.checkIn)
	p("POST /events/{id}/guests", leaders, s.createGuest)
	p("DELETE /events/{id}/checkins/{checkinID}", leaders, s.deleteCheckIn)
	p("GET /events/{id}/report", projectLeads, s.getReport)
	p("PUT /events/{id}/report", projectLeads, s.saveReport)

	p("GET /events/{id}/staff", leaders, s.listStaff)
	p("GET /events/{id}/payout", projectLeads, s.listPayout)
	p("PUT /events/{id}/payout/{userId}", projectLeads, s.signPayout)
	p("DELETE /events/{id}/payout/{userId}", projectLeads, s.unsignPayout)

	p("GET /team", leaders, s.listTeam)
	p("PUT /team/{id}/function", admins, s.setMemberFunction)

	p("GET /functions", projectLeads, s.listFunctions)
	p("POST /functions", admins, s.createFunction)
	p("PUT /functions/{id}", admins, s.updateFunction)
	p("DELETE /functions/{id}", admins, s.deleteFunction)

	p("GET /emergency-access", admins, s.getEmergencyAccess)
	p("PUT /emergency-access", admins, s.setEmergencyAccess)

	// Teilnehmer: nur Mitglieder dieses Projekts. Mitarbeiter suchen hier zum manuellen Einchecken.
	p("GET /members", leaders, s.listMembers)
	p("PUT /members/{id}/role", admins, s.setMemberRole)
	p("PUT /members/{id}/status", admins, s.setMemberStatus)
	p("GET /members/{id}/emergency-contact", leaders, s.getEmergencyContact)
	p("PUT /members/{id}/sanction", projectLeads, s.setSanction)
	p("DELETE /members/{id}/sanction", projectLeads, s.removeSanction)

	p("GET /settings/sanctions", admins, s.getSanctionRules)
	p("PUT /settings/sanctions", admins, s.setSanctionRules)
	p("GET /shares", admins, s.listShares)
	p("POST /shares/{partner}/consent", admins, s.consentShare)
	p("POST /shares/{partner}/decline", admins, s.declineShare)
	p("DELETE /shares/{partner}", admins, s.endShare)

	p("GET /locations", anyRole, s.listLocations)
	p("POST /locations", admins, s.createLocation)
	p("PUT /locations/{id}", admins, s.updateLocation)

	p("GET /time-entries", projectLeads, s.listTimeEntries)
	p("POST /time-entries", projectLeads, s.createTimeEntry)
	p("DELETE /time-entries/{id}", projectLeads, s.deleteTimeEntry)
	p("GET /time-entries/export.csv", admins, s.exportTimeEntries)

	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, errNotFound)
	})
	if s.cfg.StaticDir != "" {
		m.Handle("/", spaHandler(s.cfg.StaticDir))
	}
}

type handlerFunc func(w http.ResponseWriter, r *http.Request, u *User) error

type memberHandler func(w http.ResponseWriter, r *http.Request, m *Member) error

// route registriert einen Handler, der einen angemeldeten Benutzer verlangt (unabhängig vom Projekt).
func (s *Server) route(pattern string, h handlerFunc) {
	s.handle(pattern, false, h)
}

// adminRoute registriert einen Handler der globalen Administration (nur globale Admins).
func (s *Server) adminRoute(pattern string, h handlerFunc) {
	s.handle(pattern, true, h)
}

func (s *Server) handle(pattern string, globalAdmin bool, h handlerFunc) {
	s.protectedRoutes = append(s.protectedRoutes, pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		u, err := s.sessionUser(r)
		if err == nil && globalAdmin && !u.GlobalAdmin {
			err = errForbidden
		}
		if err == nil {
			err = h(w, r, u)
		}
		if err != nil {
			writeError(w, err)
		}
	})
}

// projectRoute registriert einen Handler im Kontext eines Projekts ({project} im Pfad). Verlangt eine
// Mitgliedschaft mit einer der Rollen (nil = alle). Globale Admins sind in jedem Projekt Admin.
func (s *Server) projectRoute(pattern string, roles []string, h memberHandler) {
	s.protectedRoutes = append(s.protectedRoutes, pattern)
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		u, err := s.sessionUser(r)
		if err != nil {
			writeError(w, err)
			return
		}
		m, err := s.loadMember(r.Context(), r.PathValue("project"), u)
		if err == nil && roles != nil && !slices.Contains(roles, m.Role) {
			err = errForbidden
		}
		if err == nil {
			err = h(w, r, m)
		}
		if err != nil {
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

func noContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// validUUID prüft das Format einer ID (z.B. Benutzer). IDs werden in SQL direkt mit der uuid-Spalte
// verglichen, damit der Index genutzt wird. Ungültige IDs würden dort einen Fehler auslösen.
func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case !strings.ContainsRune("0123456789abcdefABCDEF", c):
			return false
		}
	}
	return true
}

// pathUUID liest eine ID aus dem Pfad. Ungültige IDs gelten als nicht gefunden.
func pathUUID(r *http.Request, name string) (string, error) {
	id := r.PathValue(name)
	if !validUUID(id) {
		return "", errNotFound
	}
	return id, nil
}

func pathInt(r *http.Request, name string) (int, error) {
	n, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		return 0, errNotFound
	}
	return n, nil
}

func isProjectLead(m *Member) bool { return slices.Contains(projectLeads, m.Role) }

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
