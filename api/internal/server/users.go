package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tribock/midnightsports/api/internal/qr"
)

type User struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	BirthDate         string `json:"birthDate"`
	Street            string `json:"street"`
	Zip               string `json:"zip"`
	City              string `json:"city"`
	EmergencyName     string `json:"emergencyName"`
	EmergencyPhone    string `json:"emergencyPhone"`
	EmergencyRelation string `json:"emergencyRelation"`
	ProfileComplete   bool   `json:"profileComplete"`
}

const userColumns = `u.id, u.email, u.role, u.first_name, u.last_name,
	COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.street, u.zip, u.city,
	u.emergency_name, u.emergency_phone, u.emergency_relation`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.FirstName, &u.LastName, &u.BirthDate,
		&u.Street, &u.Zip, &u.City, &u.EmergencyName, &u.EmergencyPhone, &u.EmergencyRelation)
	if err != nil {
		return nil, err
	}
	u.ProfileComplete = !slices.Contains([]string{u.FirstName, u.LastName, u.BirthDate, u.Street,
		u.Zip, u.City, u.EmergencyName, u.EmergencyPhone, u.EmergencyRelation}, "")
	return &u, nil
}

func (u *User) Name() string { return strings.TrimSpace(u.FirstName + " " + u.LastName) }

func (s *Server) getMe(w http.ResponseWriter, r *http.Request, u *User) error {
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request, u *User) error {
	var p User
	if err := decode(r, &p); err != nil {
		return err
	}
	fields := []*string{&p.FirstName, &p.LastName, &p.BirthDate, &p.Street, &p.Zip, &p.City,
		&p.EmergencyName, &p.EmergencyPhone, &p.EmergencyRelation}
	for _, f := range fields {
		*f = strings.TrimSpace(*f)
		if *f == "" {
			return errorf(http.StatusBadRequest, "Bitte alle Felder ausfüllen")
		}
	}
	birth, err := time.Parse(time.DateOnly, p.BirthDate)
	if err != nil || birth.After(time.Now()) || birth.Year() < 1900 {
		return errorf(http.StatusBadRequest, "Ungültiges Geburtsdatum")
	}

	updated, err := scanUser(s.db.QueryRow(r.Context(),
		`UPDATE users u SET first_name = $2, last_name = $3, birth_date = $4, street = $5, zip = $6, city = $7,
		 emergency_name = $8, emergency_phone = $9, emergency_relation = $10
		 WHERE id = $1 RETURNING `+userColumns,
		u.ID, p.FirstName, p.LastName, birth, p.Street, p.Zip, p.City,
		p.EmergencyName, p.EmergencyPhone, p.EmergencyRelation))
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
	w.WriteHeader(http.StatusNoContent)
	return nil
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
}

// listUsers sucht Benutzer nach Name oder Email, optional gefiltert nach Rolle.
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, u *User) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	rows, err := s.db.Query(r.Context(),
		`SELECT id, email, role, first_name, last_name, COALESCE(to_char(birth_date, 'YYYY-MM-DD'), '')
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
		err := row.Scan(&s.ID, &s.Email, &s.Role, &s.FirstName, &s.LastName, &s.BirthDate)
		return s, err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, users)
	return nil
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, u *User) error {
	user, err := scanUser(s.db.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM users u WHERE u.id::text = $1`, r.PathValue("id")))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, user)
	return nil
}

func (s *Server) setRole(w http.ResponseWriter, r *http.Request, u *User) error {
	var req struct{ Role string }
	if err := decode(r, &req); err != nil {
		return err
	}
	if !slices.Contains([]string{RoleAdmin, RoleProjektleiter, RoleTeilnehmer}, req.Role) {
		return errorf(http.StatusBadRequest, "Ungültige Rolle")
	}
	id := r.PathValue("id")
	if id == u.ID && req.Role != RoleAdmin {
		return errorf(http.StatusBadRequest, "Die eigene Admin-Rolle kann nicht entzogen werden")
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE users SET role = $2 WHERE id::text = $1`, id, req.Role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
