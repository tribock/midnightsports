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

type User struct {
	ID                string            `json:"id"`
	Email             string            `json:"email"`
	Role              string            `json:"role"`
	FirstName         string            `json:"firstName"`
	LastName          string            `json:"lastName"`
	BirthDate         string            `json:"birthDate"`
	Gender            Gender            `json:"gender"`
	Status            UserStatus        `json:"status"`
	Street            string            `json:"street"`
	Zip               string            `json:"zip"`
	City              string            `json:"city"`
	EmergencyName     string            `json:"emergencyName"`
	EmergencyPhone    string            `json:"emergencyPhone"`
	EmergencyRelation EmergencyRelation `json:"emergencyRelation"`
	// Guest: ohne Login erfasst, Email freiwillig
	Guest bool `json:"guest"`
	// EmergencyAccess: darf Notfallkontakte anderer Benutzer einsehen
	EmergencyAccess bool `json:"emergencyAccess"`
	ProfileComplete bool `json:"profileComplete"`
}

const userColumns = `u.id, COALESCE(u.email, ''), u.role, u.first_name, u.last_name,
	COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.gender, u.status, u.street, u.zip, u.city,
	u.emergency_name, u.emergency_phone, u.emergency_relation, u.guest,
	u.role = 'admin'
	OR EXISTS (SELECT 1 FROM emergency_access_roles ar WHERE ar.role = u.role)
	OR EXISTS (SELECT 1 FROM emergency_access_functions af WHERE af.function_id = u.function_id)`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.FirstName, &u.LastName, &u.BirthDate, &u.Gender, &u.Status,
		&u.Street, &u.Zip, &u.City, &u.EmergencyName, &u.EmergencyPhone, &u.EmergencyRelation, &u.Guest, &u.EmergencyAccess)
	if err != nil {
		return nil, err
	}
	// Strasse ist freiwillig
	u.ProfileComplete = !slices.Contains([]string{u.FirstName, u.LastName, u.BirthDate, string(u.Gender), string(u.Status),
		u.Zip, u.City, u.EmergencyName, u.EmergencyPhone, string(u.EmergencyRelation)}, "")
	return &u, nil
}

func (u *User) Name() string { return strings.TrimSpace(u.FirstName + " " + u.LastName) }

// visibleTo entfernt den Notfallkontakt, wenn der Betrachter keinen Zugriff darauf hat.
func (u *User) visibleTo(viewer *User) *User {
	if viewer.EmergencyAccess || viewer.ID == u.ID {
		return u
	}
	c := *u
	c.EmergencyName, c.EmergencyPhone, c.EmergencyRelation = "", "", ""
	return &c
}

func (s *Server) getMe(w http.ResponseWriter, r *http.Request, u *User) error {
	writeJSON(w, http.StatusOK, u)
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
	if !p.Status.Valid() {
		return time.Time{}, errorf(http.StatusBadRequest, "Bitte Status auswählen")
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

// updateProfile speichert das Profil eines Benutzers aus dem Request. currentStatus ist der bisherige Status,
// den nur Admins auf Mitarbeiter setzen oder davon ändern dürfen.
// Die Email kann nur bei Gästen geändert werden, sonst ist es die Email des Google-Kontos.
func (s *Server) updateProfile(r *http.Request, editor *User, id string, currentStatus UserStatus) (*User, error) {
	var p User
	if err := decode(r, &p); err != nil {
		return nil, err
	}
	if err := checkStatusChange(editor, currentStatus, p.Status); err != nil {
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
	updated, err := scanUser(s.db.QueryRow(r.Context(),
		`UPDATE users u SET first_name = $2, last_name = $3, birth_date = $4, street = $5, zip = $6, city = $7,
		 emergency_name = $8, emergency_phone = $9, emergency_relation = $10, gender = $11, status = $12,
		 email = CASE WHEN u.guest THEN $13 ELSE u.email END
		 WHERE u.id = $1 RETURNING `+userColumns,
		id, p.FirstName, p.LastName, birth, p.Street, p.Zip, p.City,
		p.EmergencyName, p.EmergencyPhone, string(p.EmergencyRelation), string(p.Gender), string(p.Status), email))
	if isUniqueViolation(err) {
		return nil, errorf(http.StatusConflict, "Es gibt bereits einen Benutzer mit dieser Email")
	}
	return updated, err
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, u *User) error {
	updated, err := s.updateProfile(r, u, u.ID, u.Status)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

// updateUser speichert das Profil eines anderen Benutzers (nur Admins), z.B. um einem Gast eine Email zu geben.
// Meldet sich der Gast danach mit dieser Email an, übernimmt er das Konto.
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	// Nur Admins: der Status darf frei gewählt werden
	updated, err := s.updateProfile(r, u, id, "")
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

// deleteMe löscht das Konto. Check-ins bleiben anonymisiert erhalten (user_id wird NULL).
func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request, u *User) error {
	if _, err := s.db.Exec(r.Context(), `DELETE FROM users WHERE id = $1`, u.ID); err != nil {
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

type userSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	BirthDate string `json:"birthDate"`
	Guest     bool   `json:"guest"`
}

// listUsers sucht Benutzer nach Name oder Email, optional gefiltert nach Rolle.
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, u *User) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	rows, err := s.db.Query(r.Context(),
		`SELECT id, COALESCE(email, ''), role, first_name, last_name, COALESCE(to_char(birth_date, 'YYYY-MM-DD'), ''), guest
		 FROM users
		 WHERE ($1 = '' OR first_name || ' ' || last_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
		   AND ($2 = '' OR role = $2 OR ($2 = 'projektleiter' AND role = 'admin'))
		 ORDER BY last_name, first_name, email
		 LIMIT 100`, q, role)
	if err != nil {
		return err
	}
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (userSummary, error) {
		var s userSummary
		err := row.Scan(&s.ID, &s.Email, &s.Role, &s.FirstName, &s.LastName, &s.BirthDate, &s.Guest)
		return s, err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, users)
	return nil
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, u *User) error {
	user, err := s.loadUser(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, user.visibleTo(u))
	return nil
}

func (s *Server) setRole(w http.ResponseWriter, r *http.Request, u *User) error {
	var req struct{ Role string }
	if err := decode(r, &req); err != nil {
		return err
	}
	if !slices.Contains(allRoles, req.Role) {
		return errorf(http.StatusBadRequest, "Ungültige Rolle")
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	if id == u.ID && req.Role != RoleAdmin {
		return errorf(http.StatusBadRequest, "Die eigene Admin-Rolle kann nicht entzogen werden")
	}
	if err := s.execOne(r.Context(), `UPDATE users SET role = $2 WHERE id = $1`, id, req.Role); err != nil {
		return err
	}
	return noContent(w)
}

// deleteUser löscht einen Benutzer (nur Admins). Check-ins bleiben anonymisiert erhalten.
// Das eigene Konto wird über das Profil gelöscht.
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	if id == u.ID {
		return errorf(http.StatusBadRequest, "Das eigene Konto bitte über das Profil löschen")
	}
	if err := s.execOne(r.Context(), `DELETE FROM users WHERE id = $1`, id); err != nil {
		return err
	}
	return noContent(w)
}

type emergencyContact struct {
	Name     string            `json:"name"`
	Phone    string            `json:"phone"`
	Relation EmergencyRelation `json:"relation"`
}

// getEmergencyContact liefert den Notfallkontakt eines Benutzers. Erlaubt für Personen, deren Rolle
// oder Funktion Zugriff auf Notfallkontakte hat (Administration, Standard: Projektleiter). Admins immer.
func (s *Server) getEmergencyContact(w http.ResponseWriter, r *http.Request, u *User) error {
	if !u.EmergencyAccess {
		return errForbidden
	}
	id, err := pathUUID(r, "id")
	if err != nil {
		return err
	}
	var c emergencyContact
	err = s.db.QueryRow(r.Context(),
		`SELECT emergency_name, emergency_phone, emergency_relation FROM users WHERE id = $1`,
		id).Scan(&c.Name, &c.Phone, &c.Relation)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, c)
	return nil
}
