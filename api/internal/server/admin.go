package server

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- Projekte (globale Admins) ---

// AdminProject ist ein Projekt in der globalen Administration, mit Admins und geteilten Verwarnungen.
type AdminProject struct {
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Region      string         `json:"region"`
	Description string         `json:"description"`
	HasLogo     bool           `json:"hasLogo"`
	Pending     bool           `json:"pending"`
	MemberCount int            `json:"memberCount"`
	Admins      []string       `json:"admins"`
	Shares      []ProjectShare `json:"shares"`
	CreatedAt   time.Time      `json:"createdAt"`
}

// ProjectShare ist das Teilen der Verwarnungen und Sperren mit einem Partnerprojekt.
type ProjectShare struct {
	PartnerSlug string `json:"partnerSlug"`
	PartnerName string `json:"partnerName"`
	// Outgoing: dieses Projekt hat angefragt, der Partner muss einwilligen
	Outgoing bool `json:"outgoing"`
	// State: "ausstehend", "eingewilligt" oder "abgelehnt"
	State       string     `json:"state"`
	ConsentedAt *time.Time `json:"consentedAt"`
}

const shareState = `CASE WHEN sh.consented_at IS NOT NULL THEN 'eingewilligt'
	WHEN sh.declined_at IS NOT NULL THEN 'abgelehnt' ELSE 'ausstehend' END`

// loadShares liefert die geteilten Verwarnungen eines Projekts in beide Richtungen.
func loadShares(ctx context.Context, q dbtx, projectID int) ([]ProjectShare, error) {
	rows, err := q.Query(ctx,
		`SELECT p.slug, p.name, true, `+shareState+`, sh.consented_at
		 FROM project_shares sh JOIN projects p ON p.id = sh.partner_id WHERE sh.project_id = $1
		 UNION ALL
		 SELECT p.slug, p.name, false, `+shareState+`, sh.consented_at
		 FROM project_shares sh JOIN projects p ON p.id = sh.project_id WHERE sh.partner_id = $1
		 ORDER BY 2`, projectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[ProjectShare])
}

func (s *Server) listAdminProjects(w http.ResponseWriter, r *http.Request, u *User) error {
	ctx := r.Context()
	rows, err := s.db.Query(ctx,
		`SELECT p.id, p.slug, p.name, p.region, p.description, p.logo IS NOT NULL, `+projectPending+`,
		        (SELECT count(*) FROM project_members m WHERE m.project_id = p.id),
		        COALESCE((SELECT array_agg(COALESCE(NULLIF(trim(au.first_name || ' ' || au.last_name), ''), au.email)
		                                   ORDER BY au.last_name, au.email)
		                  FROM project_members am JOIN users au ON au.id = am.user_id
		                  WHERE am.project_id = p.id AND am.role = 'admin'), '{}'),
		        p.created_at
		 FROM projects p ORDER BY p.name`)
	if err != nil {
		return err
	}
	type row struct {
		id int
		AdminProject
	}
	list, err := pgx.CollectRows(rows, func(rw pgx.CollectableRow) (row, error) {
		var x row
		err := rw.Scan(&x.id, &x.Slug, &x.Name, &x.Region, &x.Description, &x.HasLogo, &x.Pending,
			&x.MemberCount, &x.Admins, &x.CreatedAt)
		return x, err
	})
	if err != nil {
		return err
	}
	result := make([]AdminProject, len(list))
	for i, x := range list {
		if x.Shares, err = loadShares(ctx, s.db, x.id); err != nil {
			return err
		}
		result[i] = x.AdminProject
	}
	writeJSON(w, http.StatusOK, result)
	return nil
}

// projectInput ist das Onboarding eines Projekts: Stammdaten, Logo, Projekt-Admin, Funktionen mit Lohn,
// ein erster Veranstaltungsort und die Projekte, mit denen Verwarnungen und Sperren geteilt werden.
type projectInput struct {
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Region      string     `json:"region"`
	Description string     `json:"description"`
	Logo        *string    `json:"logo"`
	AdminEmail  string     `json:"adminEmail"`
	Functions   []Function `json:"functions"`
	Location    Location   `json:"location"`
	ShareWith   []string   `json:"shareWith"`
}

// validateProjectInput prüft das Onboarding und normalisiert die Eingaben.
func validateProjectInput(in *projectInput) error {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Name = strings.TrimSpace(in.Name)
	in.Region = strings.TrimSpace(in.Region)
	in.Description = strings.TrimSpace(in.Description)
	in.AdminEmail = strings.ToLower(strings.TrimSpace(in.AdminEmail))
	in.Location.Name = strings.TrimSpace(in.Location.Name)
	switch {
	case in.Name == "":
		return errorf(http.StatusBadRequest, "Name fehlt")
	case !slugPattern.MatchString(in.Slug) || len(in.Slug) > 40:
		return errorf(http.StatusBadRequest, "Kurzname: nur Kleinbuchstaben, Ziffern und Bindestriche (z.B. pleno-frauenfeld)")
	case in.Region == "":
		return errorf(http.StatusBadRequest, "Region fehlt (z.B. Aadorf)")
	case in.Location.Name == "":
		return errorf(http.StatusBadRequest, "Bitte einen ersten Veranstaltungsort angeben")
	case len(in.Functions) == 0:
		return errorf(http.StatusBadRequest, "Bitte mindestens eine Funktion angeben")
	}
	if _, err := mail.ParseAddress(in.AdminEmail); err != nil {
		return errorf(http.StatusBadRequest, "Ungültige Email des Projekt-Admins")
	}
	var names []string
	for i := range in.Functions {
		f := &in.Functions[i]
		f.Name = strings.TrimSpace(f.Name)
		if f.Name == "" {
			return errorf(http.StatusBadRequest, "Funktion ohne Bezeichnung")
		}
		if slices.Contains(names, strings.ToLower(f.Name)) {
			return errorf(http.StatusBadRequest, "Funktion "+f.Name+" ist doppelt")
		}
		if f.Salary != nil && *f.Salary < 0 {
			return errorf(http.StatusBadRequest, "Lohn darf nicht negativ sein")
		}
		names = append(names, strings.ToLower(f.Name))
	}
	if slices.Contains(in.ShareWith, in.Slug) {
		return errorf(http.StatusBadRequest, "Ein Projekt kann nicht mit sich selbst teilen")
	}
	return nil
}

// createProject legt ein Projekt mit allem an, was ein Projekt braucht (wie Aadorf): Funktionen, Zugriff auf
// Notfallkontakte (Projektleiter), Standardort und Projekt-Admin. Gibt es noch keine Person mit der Email des
// Admins, wird sie angelegt und übernommen, sobald sie sich mit dieser Email anmeldet.
// Mit Teilen-Anfragen bleibt das Projekt ausstehend, bis die Partner eingewilligt haben.
func (s *Server) createProject(w http.ResponseWriter, r *http.Request, u *User) error {
	var in projectInput
	if err := decode(r, &in); err != nil {
		return err
	}
	if err := validateProjectInput(&in); err != nil {
		return err
	}
	var logo []byte
	var logoType *string
	if in.Logo != nil && *in.Logo != "" {
		img, t, err := parseLogo(*in.Logo)
		if err != nil {
			return err
		}
		logo, logoType = img, &t
	}

	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var projectID int
	err = tx.QueryRow(ctx,
		`INSERT INTO projects (slug, name, region, description, logo, logo_type) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		in.Slug, in.Name, in.Region, in.Description, logo, logoType).Scan(&projectID)
	if isUniqueViolation(err) {
		return errorf(http.StatusConflict, "Es gibt bereits ein Projekt mit diesem Kurznamen")
	}
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, f := range in.Functions {
		batch.Queue(`INSERT INTO functions (project_id, name, salary) VALUES ($1, $2, $3)`, projectID, f.Name, f.Salary)
	}
	batch.Queue(`INSERT INTO emergency_access_roles (project_id, role) VALUES ($1, 'projektleiter')`, projectID)
	l := in.Location
	batch.Queue(`INSERT INTO locations (project_id, name, street, zip, city, is_default) VALUES ($1, $2, $3, $4, $5, true)`,
		projectID, l.Name, strings.TrimSpace(l.Street), strings.TrimSpace(l.Zip), strings.TrimSpace(l.City))
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}

	admin, err := s.userByEmail(ctx, tx, in.AdminEmail)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, 'admin')`, projectID, admin.ID); err != nil {
		return err
	}

	if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &projectID, Action: actionProjectCreated,
		Details: map[string]any{"name": in.Name, "kurzname": in.Slug}}); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &projectID, Action: actionRoleChanged, Target: admin,
		Details: map[string]any{"vorher": "", "nachher": RoleAdmin}}); err != nil {
		return err
	}
	for _, slug := range in.ShareWith {
		var partnerID int
		if err := tx.QueryRow(ctx, `SELECT id FROM projects WHERE slug = $1`, slug).Scan(&partnerID); err != nil {
			return errorf(http.StatusBadRequest, "Unbekanntes Partnerprojekt "+slug)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO project_shares (project_id, partner_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			projectID, partnerID); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &projectID, Action: actionShareRequested,
			Details: map[string]any{"partner": slug}}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]string{"slug": in.Slug})
	return nil
}

// userByEmail liefert die Person mit dieser Email oder legt sie ohne Profil an. Meldet sie sich mit Google
// oder Apple und derselben Email an, übernimmt sie das Konto.
func (s *Server) userByEmail(ctx context.Context, q dbtx, email string) (*User, error) {
	u, err := scanUser(q.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.email = $1`, email))
	if !errors.Is(err, pgx.ErrNoRows) {
		return u, err
	}
	return scanUser(q.QueryRow(ctx, `INSERT INTO users AS u (email) VALUES ($1) RETURNING `+userColumns, email))
}

// updateProject ändert Name, Region, Beschreibung und Logo (globale Admins). logo: null = unverändert,
// "" = Standardlogo, sonst das neue Bild.
func (s *Server) updateProject(w http.ResponseWriter, r *http.Request, u *User) error {
	var in projectInput
	if err := decode(r, &in); err != nil {
		return err
	}
	in.Name, in.Region, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Region), strings.TrimSpace(in.Description)
	if in.Name == "" || in.Region == "" {
		return errorf(http.StatusBadRequest, "Name und Region sind Pflicht")
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var projectID int
	if err := tx.QueryRow(ctx,
		`UPDATE projects SET name = $2, region = $3, description = $4 WHERE slug = $1 RETURNING id`,
		r.PathValue("project"), in.Name, in.Region, in.Description).Scan(&projectID); err != nil {
		return err
	}
	details := map[string]any{"name": in.Name}
	if in.Logo != nil {
		var logo []byte
		var logoType *string
		if *in.Logo != "" {
			img, t, err := parseLogo(*in.Logo)
			if err != nil {
				return err
			}
			logo, logoType = img, &t
		}
		if _, err := tx.Exec(ctx, `UPDATE projects SET logo = $2, logo_type = $3 WHERE id = $1`, projectID, logo, logoType); err != nil {
			return err
		}
		details["logo"] = logo != nil
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &projectID, Action: actionProjectUpdated, Details: details}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// removeShare entfernt eine Teilen-Anfrage oder ein bestehendes Teilen (globale Admins), z.B. nach einer Ablehnung.
// Ein ausstehendes Projekt wird damit aktiv, wenn keine weitere Anfrage offen ist.
func (s *Server) removeShare(w http.ResponseWriter, r *http.Request, u *User) error {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var projectID int
	err = tx.QueryRow(ctx,
		`DELETE FROM project_shares sh USING projects p, projects q
		 WHERE p.slug = $1 AND q.slug = $2
		   AND ((sh.project_id = p.id AND sh.partner_id = q.id) OR (sh.project_id = q.id AND sh.partner_id = p.id))
		 RETURNING p.id`, r.PathValue("project"), r.PathValue("partner")).Scan(&projectID)
	if err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: u, ProjectID: &projectID, Action: actionShareRemoved,
		Details: map[string]any{"partner": r.PathValue("partner")}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// --- Benutzer (globale Admins) ---

// AdminUser ist eine Person in der globalen Benutzerliste, mit ihren Projekten.
type AdminUser struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	FirstName   string   `json:"firstName"`
	LastName    string   `json:"lastName"`
	BirthDate   string   `json:"birthDate"`
	Guest       bool     `json:"guest"`
	GlobalAdmin bool     `json:"globalAdmin"`
	Projects    []string `json:"projects"`
}

// listAdminUsers sucht Personen über alle Projekte nach Name oder Email.
func (s *Server) listAdminUsers(w http.ResponseWriter, r *http.Request, u *User) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := s.db.Query(r.Context(),
		`SELECT u.id::text, COALESCE(u.email, ''), u.first_name, u.last_name,
		        COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.guest, u.global_admin,
		        COALESCE((SELECT array_agg(p.name ORDER BY p.name)
		                  FROM project_members m JOIN projects p ON p.id = m.project_id WHERE m.user_id = u.id), '{}')
		 FROM users u
		 WHERE $1 = '' OR u.first_name || ' ' || u.last_name ILIKE '%' || $1 || '%' OR u.email ILIKE '%' || $1 || '%'
		 ORDER BY u.last_name, u.first_name, u.email
		 LIMIT 100`, q)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[AdminUser])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

// getAdminUser liefert das vollständige Profil einer Person. Der Zugriff wird protokolliert.
func (s *Server) getAdminUser(w http.ResponseWriter, r *http.Request, u *User) error {
	target, err := s.loadUser(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	if err := s.audit(r.Context(), s.db, auditEntry{Actor: u, Action: actionProfileViewed, Target: target}); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, target)
	return nil
}

// updateAdminUser ändert das Profil einer Person (nur globale Admins, z.B. Email für einen Gast).
// Protokolliert werden die geänderten Felder, nicht die Werte.
func (s *Server) updateAdminUser(w http.ResponseWriter, r *http.Request, u *User) error {
	ctx := r.Context()
	old, err := s.loadUser(ctx, r.PathValue("id"))
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	updated, err := s.updateProfile(r, tx, old.ID)
	if err != nil {
		return err
	}
	if changed := changedFields(old, updated); len(changed) > 0 {
		if err := s.audit(ctx, tx, auditEntry{Actor: u, Action: actionProfileUpdated, Target: updated,
			Details: map[string]any{"felder": changed}}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

// deleteAdminUser löscht eine Person (nur globale Admins). Das eigene Konto wird über das Profil gelöscht.
func (s *Server) deleteAdminUser(w http.ResponseWriter, r *http.Request, u *User) error {
	target, err := s.loadUser(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	if target.ID == u.ID {
		return errorf(http.StatusBadRequest, "Das eigene Konto bitte über das Profil löschen")
	}
	if err := s.deleteUserData(r.Context(), u, target); err != nil {
		return err
	}
	return noContent(w)
}

// deleteUserData löscht eine Person mit allen Mitgliedschaften. Check-ins bleiben anonymisiert erhalten,
// im Audit-Protokoll wird der Name durch "Gelöschtes Konto" ersetzt.
func (s *Server) deleteUserData(ctx context.Context, actor, target *User) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	action := actionUserDeleted
	if actor.ID == target.ID {
		action = actionAccountDeleted
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: actor, Action: action, Target: target}); err != nil {
		return err
	}
	if err := anonymizeAudit(ctx, tx, target.ID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, target.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return tx.Commit(ctx)
}
