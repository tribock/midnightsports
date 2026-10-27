package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// MemberSummary ist ein Mitglied in der Teilnehmerliste eines Projekts.
type MemberSummary struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Status    UserStatus `json:"status"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	BirthDate string     `json:"birthDate"`
	Guest     bool       `json:"guest"`
	JoinedAt  time.Time  `json:"joinedAt"`
	Sanctions []Sanction `json:"sanctions"`
}

// listMembers sucht Mitglieder des Projekts nach Name oder Email, optional gefiltert nach Rolle.
// Projekt-Admins sehen nur die Personen, die sich für ihr Projekt angemeldet haben.
func (s *Server) listMembers(w http.ResponseWriter, r *http.Request, m *Member) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	rows, err := s.db.Query(r.Context(),
		`SELECT u.id::text, COALESCE(u.email, ''), pm.role, pm.status, u.first_name, u.last_name,
		        COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.guest, pm.joined_at
		 FROM project_members pm
		 JOIN users u ON u.id = pm.user_id
		 WHERE pm.project_id = $1
		   AND ($2 = '' OR u.first_name || ' ' || u.last_name ILIKE '%' || $2 || '%' OR u.email ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR pm.role = $3 OR ($3 = 'projektleiter' AND pm.role = 'admin'))
		 ORDER BY u.last_name, u.first_name, u.email
		 LIMIT 100`, m.Project.ID, q, role)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (MemberSummary, error) {
		var ms MemberSummary
		err := row.Scan(&ms.ID, &ms.Email, &ms.Role, &ms.Status, &ms.FirstName, &ms.LastName, &ms.BirthDate,
			&ms.Guest, &ms.JoinedAt)
		return ms, err
	})
	if err != nil {
		return err
	}
	ids := make([]string, len(list))
	for i, ms := range list {
		ids[i] = ms.ID
	}
	sanctions, err := s.visibleSanctions(r.Context(), s.db, m.Project.ID, ids)
	if err != nil {
		return err
	}
	for i := range list {
		list[i].Sanctions = sanctions[list[i].ID]
		if list[i].Sanctions == nil {
			list[i].Sanctions = []Sanction{}
		}
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

// setMemberRole ändert die Rolle eines Mitglieds im Projekt (Projekt-Admins).
func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct{ Role string }
	if err := decode(r, &req); err != nil {
		return err
	}
	if !slices.Contains(allRoles, req.Role) {
		return errorf(http.StatusBadRequest, "Ungültige Rolle")
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	if target.ID == m.ID && req.Role != RoleAdmin {
		return errorf(http.StatusBadRequest, "Die eigene Admin-Rolle kann nicht entzogen werden")
	}
	return s.changeMembership(w, r, m, target, "role", req.Role, actionRoleChanged)
}

// setMemberStatus ändert den Status eines Mitglieds (Projekt-Admins, z.B. Mitarbeiter vergeben).
func (s *Server) setMemberStatus(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct{ Status UserStatus }
	if err := decode(r, &req); err != nil {
		return err
	}
	if !req.Status.Valid() {
		return errorf(http.StatusBadRequest, "Ungültiger Status")
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	return s.changeMembership(w, r, m, target, "status", string(req.Status), actionStatusChanged)
}

// changeMembership setzt Rolle oder Status (column) eines Mitglieds und protokolliert alten und neuen Wert.
func (s *Server) changeMembership(w http.ResponseWriter, r *http.Request, m *Member, target *User, column, value, action string) error {
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old string
	// column ist "role" oder "status" aus dem Code, nie aus dem Request
	if err := tx.QueryRow(ctx,
		`SELECT `+column+` FROM project_members WHERE project_id = $1 AND user_id = $2 FOR UPDATE`,
		m.Project.ID, target.ID).Scan(&old); err != nil {
		return err
	}
	if old == value {
		return noContent(w)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE project_members SET `+column+` = $3 WHERE project_id = $1 AND user_id = $2`,
		m.Project.ID, target.ID, value); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: action, Target: target,
		Details: map[string]any{"vorher": old, "nachher": value}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

type emergencyContact struct {
	Name     string            `json:"name"`
	Phone    string            `json:"phone"`
	Relation EmergencyRelation `json:"relation"`
}

// getEmergencyContact liefert den Notfallkontakt eines Mitglieds. Erlaubt für Personen, deren Rolle oder Funktion
// im Projekt Zugriff hat (Administration, Standard: Projektleiter), Projekt-Admins und globale Admins.
// Jeder Zugriff wird protokolliert.
func (s *Server) getEmergencyContact(w http.ResponseWriter, r *http.Request, m *Member) error {
	if !m.EmergencyAccess {
		return errForbidden
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	if err := s.audit(r.Context(), s.db, auditEntry{Actor: m.User, ProjectID: &m.Project.ID,
		Action: actionEmergencyViewed, Target: target}); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, emergencyContact{target.EmergencyName, target.EmergencyPhone, target.EmergencyRelation})
	return nil
}
