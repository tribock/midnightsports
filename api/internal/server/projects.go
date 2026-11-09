package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Project ist ein Mandant (z.B. Midnightsports Aadorf). Alle Events, Orte, Funktionen und Mitglieder gehören
// zu genau einem Projekt.
type Project struct {
	ID          int    `json:"-"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Region      string `json:"region"`
	Description string `json:"description"`
	HasLogo     bool   `json:"hasLogo"`
	// Pending: wartet auf die Einwilligung eines Partners zum Teilen der Verwarnungen (nur globale Admins)
	Pending bool `json:"pending"`
	sanctionRules
}

// Member ist der angemeldete Benutzer im Kontext eines Projekts (Routen unter /api/p/{project}/).
// Globale Admins sind in jedem Projekt Admin, auch ohne Mitgliedschaft.
type Member struct {
	*User
	Project    *Project   `json:"project"`
	Role       string     `json:"role"`
	Status     UserStatus `json:"status"`
	FunctionID *int       `json:"functionId"`
	// EmergencyAccess: darf Notfallkontakte der Mitglieder einsehen (Rolle oder Funktion, siehe Administration)
	EmergencyAccess bool `json:"emergencyAccess"`
	// IsMember: echte Mitgliedschaft (globale Admins ohne Mitgliedschaft: false)
	IsMember bool `json:"isMember"`
}

// projectPending: das Projekt p wartet noch auf die Einwilligung eines Partners.
const projectPending = `EXISTS (SELECT 1 FROM project_shares sh WHERE sh.project_id = p.id AND sh.consented_at IS NULL)`

const projectColumns = `p.id, p.slug, p.name, p.region, p.description, p.logo IS NOT NULL, ` + projectPending + `,
	p.ban_days, p.ban_action, p.warning_days`

func scanProject(row pgx.Row) (*Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.Region, &p.Description, &p.HasLogo, &p.Pending,
		&p.BanDays, &p.BanAction, &p.WarningDays)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// loadMember lädt Projekt und Mitgliedschaft des Benutzers in einer Abfrage.
func (s *Server) loadMember(ctx context.Context, slug string, u *User) (*Member, error) {
	if !slugPattern.MatchString(slug) {
		return nil, errNotFound
	}
	m := &Member{User: u, Project: &Project{}}
	p := m.Project
	var role, status *string
	err := s.db.QueryRow(ctx,
		`SELECT `+projectColumns+`, m.role, m.status, m.function_id,
		        -- ohne Mitgliedschaft ist m.role NULL, der Vergleich ebenfalls
		        COALESCE(m.role = 'admin'
		        OR EXISTS (SELECT 1 FROM emergency_access_roles ar WHERE ar.project_id = p.id AND ar.role = m.role)
		        OR EXISTS (SELECT 1 FROM emergency_access_functions af WHERE af.function_id = m.function_id), false)
		 FROM projects p
		 LEFT JOIN project_members m ON m.project_id = p.id AND m.user_id = $2
		 WHERE p.slug = $1`, slug, u.ID).
		Scan(&p.ID, &p.Slug, &p.Name, &p.Region, &p.Description, &p.HasLogo, &p.Pending,
			&p.BanDays, &p.BanAction, &p.WarningDays, &role, &status, &m.FunctionID, &m.EmergencyAccess)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorf(http.StatusNotFound, "Projekt nicht gefunden")
	}
	if err != nil {
		return nil, err
	}
	m.IsMember = role != nil
	if m.IsMember {
		m.Role, m.Status = *role, UserStatus(*status)
	}
	if u.GlobalAdmin {
		m.Role, m.EmergencyAccess = RoleAdmin, true
	}
	switch {
	case m.Role == "":
		return nil, errorf(http.StatusForbidden, "Du bist für dieses Projekt nicht angemeldet")
	case p.Pending && !u.GlobalAdmin:
		return nil, errorf(http.StatusConflict, "Das Projekt wartet noch auf die Einwilligung eines Partnerprojekts")
	}
	return m, nil
}

func (s *Server) getProjectMe(w http.ResponseWriter, r *http.Request, m *Member) error {
	writeJSON(w, http.StatusOK, m)
	return nil
}

// projectSearchResult ist ein Projekt in der Suche, mit der eigenen Mitgliedschaft.
type projectSearchResult struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Region      string `json:"region"`
	Description string `json:"description"`
	HasLogo     bool   `json:"hasLogo"`
	Member      bool   `json:"member"`
}

// searchProjects durchsucht die aktiven Projekte nach Name, Region, Beschreibung und Kurzname.
// Jedes Wort muss irgendwo vorkommen. Ohne Suchbegriff: alle Projekte.
func (s *Server) searchProjects(w http.ResponseWriter, r *http.Request, u *User) error {
	words := strings.Fields(strings.ToLower(r.URL.Query().Get("q")))
	rows, err := s.db.Query(r.Context(),
		`SELECT p.slug, p.name, p.region, p.description, p.logo IS NOT NULL,
		        EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = p.id AND m.user_id = $2)
		 FROM projects p
		 WHERE NOT `+projectPending+`
		   AND NOT EXISTS (
		       SELECT 1 FROM unnest($1::text[]) w
		       WHERE strpos(lower(p.name || ' ' || p.region || ' ' || p.description || ' ' || p.slug), w) = 0)
		 ORDER BY p.name
		 LIMIT 50`, words, u.ID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[projectSearchResult])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

// publicProject liefert Name und Region eines aktiven Projekts ohne Anmeldung, für die Login-Seite
// über den Projekt-Link (/?projekt=aadorf).
func (s *Server) publicProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.activeProject(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"slug": p.Slug, "name": p.Name, "region": p.Region, "hasLogo": p.HasLogo})
}

func (s *Server) activeProject(ctx context.Context, slug string) (*Project, error) {
	if !slugPattern.MatchString(slug) {
		return nil, errNotFound
	}
	p, err := scanProject(s.db.QueryRow(ctx, `SELECT `+projectColumns+` FROM projects p WHERE p.slug = $1`, slug))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && p.Pending) {
		return nil, errorf(http.StatusNotFound, "Projekt nicht gefunden")
	}
	return p, err
}

// projectLogo liefert das hochgeladene Logo (öffentlich, auch für Login-Seite und Druck). Ohne Logo: 404,
// das Frontend zeigt dann das Standardlogo. Logos nicht freigegebener Projekte nur für globale Admins.
func (s *Server) projectLogo(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("project")
	var logo []byte
	var contentType string
	var pending bool
	err := s.db.QueryRow(r.Context(),
		`SELECT p.logo, p.logo_type, `+projectPending+` FROM projects p WHERE p.slug = $1 AND p.logo IS NOT NULL`,
		slug).Scan(&logo, &contentType, &pending)
	if err == nil && pending {
		if u, uerr := s.sessionUser(r); uerr != nil || !u.GlobalAdmin {
			err = errNotFound
		}
	}
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	// Kurz cachen: ein neues Logo soll bald sichtbar sein. Nicht freigegebene nie in gemeinsamen Caches
	if pending {
		w.Header().Set("Cache-Control", "private, no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	w.Write(logo)
}

// joinProject meldet den Benutzer als Teilnehmer bei einem Projekt an (Startseite oder Projekt-Link).
// Den Status (Schüler/in aus der Region oder Gast extern) wählt die Person selbst, Mitarbeiter vergeben Projekt-Admins.
func (s *Server) joinProject(w http.ResponseWriter, r *http.Request, u *User) error {
	if !u.ProfileComplete {
		return errIncomplete
	}
	var req struct {
		Status UserStatus `json:"status"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if !req.Status.Valid() || req.Status == StatusMitarbeiter {
		return errorf(http.StatusBadRequest, "Bitte Status auswählen")
	}
	ctx := r.Context()
	p, err := s.activeProject(ctx, r.PathValue("project"))
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, status) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		p.ID, u.ID, string(req.Status))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &p.ID, Action: actionJoined, Target: u,
			Details: map[string]any{"status": req.Status}}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"slug": p.Slug})
	return nil
}

// --- Logo ---

const maxLogoSize = 512 << 10

// parseLogo prüft ein hochgeladenes Logo (base64, optional als data-URL) und liefert Bild und Typ.
// Erlaubt sind PNG, JPEG und WebP. SVG nicht, weil es Skripte enthalten kann.
func parseLogo(s string) ([]byte, string, error) {
	if i := strings.Index(s, ";base64,"); strings.HasPrefix(s, "data:") && i > 0 {
		s = s[i+len(";base64,"):]
	}
	img, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, "", errorf(http.StatusBadRequest, "Ungültiges Logo")
	}
	if len(img) > maxLogoSize {
		return nil, "", errorf(http.StatusBadRequest, "Das Logo ist zu gross (höchstens 512 KB)")
	}
	switch {
	case bytes.HasPrefix(img, pngMagic):
		return img, "image/png", nil
	case bytes.HasPrefix(img, []byte{0xFF, 0xD8, 0xFF}):
		return img, "image/jpeg", nil
	case len(img) > 12 && string(img[:4]) == "RIFF" && string(img[8:12]) == "WEBP":
		return img, "image/webp", nil
	}
	return nil, "", errorf(http.StatusBadRequest, "Das Logo muss ein PNG-, JPEG- oder WebP-Bild sein")
}
