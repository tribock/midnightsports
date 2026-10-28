package server

import (
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tribock/midnightsports/api/internal/qr"
)

// User ist eine Person, unabhängig vom Projekt. Rolle, Status und Funktion gehören zur Mitgliedschaft (Member).
type User struct {
	ID                string            `json:"id"`
	Email             string            `json:"email"`
	FirstName         string            `json:"firstName"`
	LastName          string            `json:"lastName"`
	BirthDate         string            `json:"birthDate"`
	Gender            Gender            `json:"gender"`
	Street            string            `json:"street"`
	Zip               string            `json:"zip"`
	City              string            `json:"city"`
	EmergencyName     string            `json:"emergencyName"`
	EmergencyPhone    string            `json:"emergencyPhone"`
	EmergencyRelation EmergencyRelation `json:"emergencyRelation"`
	// Guest: ohne Login erfasst, Email freiwillig
	Guest bool `json:"guest"`
	// GlobalAdmin: verwaltet Projekte, Benutzer und das Audit-Protokoll, ist in jedem Projekt Admin
	GlobalAdmin     bool `json:"globalAdmin"`
	ProfileComplete bool `json:"profileComplete"`
}

const userColumns = `u.id, COALESCE(u.email, ''), u.first_name, u.last_name,
	COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.gender, u.street, u.zip, u.city,
	u.emergency_name, u.emergency_phone, u.emergency_relation, u.guest, u.global_admin`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.BirthDate, &u.Gender,
		&u.Street, &u.Zip, &u.City, &u.EmergencyName, &u.EmergencyPhone, &u.EmergencyRelation, &u.Guest, &u.GlobalAdmin)
	if err != nil {
		return nil, err
	}
	// Strasse ist freiwillig
	u.ProfileComplete = !slices.Contains([]string{u.FirstName, u.LastName, u.BirthDate, string(u.Gender),
		u.Zip, u.City, u.EmergencyName, u.EmergencyPhone, string(u.EmergencyRelation)}, "")
	return &u, nil
}

func (u *User) Name() string { return strings.TrimSpace(u.FirstName + " " + u.LastName) }

// label ist der Name für das Audit-Protokoll, ohne Namen die Email.
func (u *User) label() string {
	if n := u.Name(); n != "" {
		return n
	}
	return u.Email
}

// visibleTo entfernt den Notfallkontakt, wenn der Betrachter keinen Zugriff darauf hat (eigene Daten immer).
func (u *User) visibleTo(viewerID string, emergencyAccess bool) *User {
	if emergencyAccess || viewerID == u.ID {
		return u
	}
	c := *u
	c.EmergencyName, c.EmergencyPhone, c.EmergencyRelation = "", "", ""
	return &c
}

// me ist der angemeldete Benutzer mit seinen Projekten (Projektauswahl auf der Startseite).
type me struct {
	*User
	Projects []memberProject `json:"projects"`
}

type memberProject struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Pending bool   `json:"pending"`
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request, u *User) error {
	rows, err := s.db.Query(r.Context(),
		`SELECT p.slug, p.name, m.role, `+projectPending+`
		 FROM project_members m JOIN projects p ON p.id = m.project_id
		 WHERE m.user_id = $1 ORDER BY p.name`, u.ID)
	if err != nil {
		return err
	}
	projects, err := pgx.CollectRows(rows, pgx.RowToStructByPos[memberProject])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, me{u, projects})
	return nil
}

// validateProfile prüft die Pflichtfelder (für Benutzer und Gäste gleich) und gibt das Geburtsdatum zurück.
func validateProfile(p *User) (time.Time, error) {
	p.Street = strings.TrimSpace(p.Street)
	required := []*string{&p.FirstName, &p.LastName, &p.BirthDate, &p.Zip, &p.City,
		&p.EmergencyName, &p.EmergencyPhone}
	for _, f := range required {
		*f = strings.TrimSpace(*f)
		if *f == "" {
			return time.Time{}, errorf(http.StatusBadRequest, "Bitte alle Pflichtfelder ausfüllen")
		}
	}
	if !p.Gender.Valid() {
		return time.Time{}, errorf(http.StatusBadRequest, "Bitte Geschlecht auswählen")
	}
	if !p.EmergencyRelation.Valid() {
		return time.Time{}, errorf(http.StatusBadRequest, "Bitte Beziehung zum Notfallkontakt auswählen")
	}
	birth, err := time.Parse(time.DateOnly, p.BirthDate)
	if err != nil || birth.After(time.Now()) || birth.Year() < 1900 {
		return time.Time{}, errorf(http.StatusBadRequest, "Ungültiges Geburtsdatum")
	}
	return birth, nil
}

// optionalEmail prüft eine freiwillige Email (Gäste). Leer ergibt nil.
func optionalEmail(email string) (*string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return nil, nil
	}
	if _, err := mail.ParseAddress(e); err != nil {
		return nil, errorf(http.StatusBadRequest, "Ungültige Email-Adresse")
	}
	return &e, nil
}

// changedFields nennt die geänderten Profilfelder (für das Audit-Protokoll, ohne die Werte selbst).
func changedFields(old, next *User) []string {
	fields := []struct {
		name string
		a, b string
	}{
		{"email", old.Email, next.Email},
		{"vorname", old.FirstName, next.FirstName},
		{"name", old.LastName, next.LastName},
		{"geburtsdatum", old.BirthDate, next.BirthDate},
		{"geschlecht", string(old.Gender), string(next.Gender)},
		{"strasse", old.Street, next.Street},
		{"plz", old.Zip, next.Zip},
		{"ort", old.City, next.City},
		{"notfallkontakt", old.EmergencyName + old.EmergencyPhone + string(old.EmergencyRelation),
			next.EmergencyName + next.EmergencyPhone + string(next.EmergencyRelation)},
	}
	var changed []string
	for _, f := range fields {
		if f.a != f.b {
			changed = append(changed, f.name)
		}
	}
	return changed
}

// updateProfile speichert das Profil einer Person aus dem Request.
// Die Email kann nur bei Gästen geändert werden, sonst ist es die Email des Google- oder Apple-Kontos.
func (s *Server) updateProfile(r *http.Request, q dbtx, id string) (*User, error) {
	var p User
	if err := decode(r, &p); err != nil {
		return nil, err
	}
	birth, err := validateProfile(&p)
	if err != nil {
		return nil, err
	}
	email, err := optionalEmail(p.Email)
	if err != nil {
		return nil, err
	}
	updated, err := scanUser(q.QueryRow(r.Context(),
		`UPDATE users u SET first_name = $2, last_name = $3, birth_date = $4, street = $5, zip = $6, city = $7,
		 emergency_name = $8, emergency_phone = $9, emergency_relation = $10, gender = $11,
		 email = CASE WHEN u.guest THEN $12 ELSE u.email END
		 WHERE u.id = $1 RETURNING `+userColumns,
		id, p.FirstName, p.LastName, birth, p.Street, p.Zip, p.City,
		p.EmergencyName, p.EmergencyPhone, string(p.EmergencyRelation), string(p.Gender), email))
	if isUniqueViolation(err) {
		return nil, errorf(http.StatusConflict, "Es gibt bereits einen Benutzer mit dieser Email")
	}
	return updated, err
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, u *User) error {
	updated, err := s.updateProfile(r, s.db, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

// deleteMe löscht das eigene Konto (Recht auf Löschung). Check-ins bleiben anonymisiert erhalten (user_id wird NULL).
func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request, u *User) error {
	if err := s.deleteUserData(r.Context(), u, u); err != nil {
		return err
	}
	http.SetCookie(w, s.cookie(sessionCookie, "", -1))
	return noContent(w)
}

func (s *Server) getMyQR(w http.ResponseWriter, r *http.Request, u *User) error {
	if !u.ProfileComplete {
		return errIncomplete
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": qr.Sign(s.cfg.QRSigningKey, u.ID)})
	return nil
}
