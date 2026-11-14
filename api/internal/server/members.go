package server

import (
	"net/http"
	"slices"
	"strings"
	"time"
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

// listMembers sucht Mitglieder des Projekts nach Name oder Email, optional gefiltert nach Rolle, seitenweise.
// Projekt-Admins sehen nur die Personen, die sich für ihr Projekt angemeldet haben.
func (s *Server) listMembers(w http.ResponseWriter, r *http.Request, m *Member) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	p := readPage(r)
	rows, err := s.db.Query(r.Context(),
		`SELECT u.id::text, COALESCE(u.email, ''), pm.role, pm.status, u.first_name, u.last_name,
		        COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), u.guest, pm.joined_at, `+pageTotal+`
		 FROM project_members pm
		 JOIN users u ON u.id = pm.user_id
		 WHERE pm.project_id = $1
		   AND ($2 = '' OR u.first_name || ' ' || u.last_name ILIKE '%' || $2 || '%' OR u.email ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR pm.role = $3 OR ($3 = 'projektleiter' AND pm.role = 'admin'))
		 ORDER BY u.last_name, u.first_name, u.email, u.id`+p.sql(4),
		append([]any{m.Project.ID, q, role}, p.args()...)...)
	if err != nil {
		return err
	}
	page, err := collectPage(rows, p, func(ms *MemberSummary) []any {
		return []any{&ms.ID, &ms.Email, &ms.Role, &ms.Status, &ms.FirstName, &ms.LastName, &ms.BirthDate,
			&ms.Guest, &ms.JoinedAt}
	})
	if err != nil {
		return err
	}
	ids := make([]string, len(page.Items))
	for i, ms := range page.Items {
		ids[i] = ms.ID
	}
	sanctions, err := s.visibleSanctions(r.Context(), s.db, m.Project.ID, ids)
	if err != nil {
		return err
	}
	for i := range page.Items {
		page.Items[i].Sanctions = sanctions[page.Items[i].ID]
		if page.Items[i].Sanctions == nil {
			page.Items[i].Sanctions = []Sanction{}
		}
	}
	writeJSON(w, http.StatusOK, page)
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

// setMemberStatus ändert den Status eines Mitglieds (Projekt-Admins, z.B. Mitarbeiter vergeben). Mitarbeiter
// brauchen eine Funktion (functionId), wer den Status Mitarbeiter verliert, verliert auch die Funktion.
func (s *Server) setMemberStatus(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct {
		Status     UserStatus `json:"status"`
		FunctionID *int       `json:"functionId"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if !req.Status.Valid() {
		return errorf(http.StatusBadRequest, "Ungültiger Status")
	}
	functionID, err := staffFunction(req.Status, req.FunctionID)
	if err != nil {
		return err
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old UserStatus
	if err := tx.QueryRow(ctx,
		`SELECT status FROM project_members WHERE project_id = $1 AND user_id = $2 FOR UPDATE`,
		m.Project.ID, target.ID).Scan(&old); err != nil {
		return err
	}
	// Bleibt jemand Mitarbeiter, ändert sich die Funktion nur über das Team
	if old == req.Status {
		return noContent(w)
	}
	var function *string
	err = tx.QueryRow(ctx,
		`UPDATE project_members SET status = $3, function_id = $4 WHERE project_id = $1 AND user_id = $2
		 RETURNING (SELECT f.name FROM functions f WHERE f.id = $4)`,
		m.Project.ID, target.ID, req.Status, functionID).Scan(&function)
	if isForeignKeyViolation(err) {
		return errorf(http.StatusBadRequest, "Unbekannte Funktion")
	}
	if err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionStatusChanged,
		Target: target, Details: map[string]any{"vorher": old, "nachher": req.Status, "funktion": function}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// changeMembership setzt die Rolle (column) eines Mitglieds und protokolliert alten und neuen Wert.
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

// // getMemberDetails liefert für alle Mitarbeiter die historie, wann registriert, Änderungen von Rolle und anzahl events, bedingt UserStatus.StatusMitarbeiter
// func (s *Server) getMemberDetails(w http.ResponseWriter, r *http.Request, caller *Member, target *Member, column, value, action string) error {
// 	ctx := r.Context()

// 	// Verify caller and target are present and both are at least a mitarbeiter
// 	if caller.User == nil {
// 		return fmt.Errorf("no caller present in request (nil)")
// 	}

// 	if caller.Status != StatusMitarbeiter {
// 		return fmt.Errorf("userstate of caller is not %s as required - current state: %s", StatusMitarbeiter, caller.Status)
// 	}

// 	tx, err := s.db.Begin(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	if target.User != nil {
// 		if target.Status != StatusMitarbeiter {
// 			return fmt.Errorf("userstate of target is not %s as required - current state: %s", StatusMitarbeiter, target.Status)
// 		}
// 	} else {
// 		return fmt.Errorf("no target present in request (nil)")
// 	}

// 	defer tx.Rollback(ctx)

// 	// get all audit events from audit table that are associated with the user

// 	var records []AuditRecord
// 	if err := tx.QueryRow(ctx,
// 		`SELECT * FROM `, "",
// 	).Scan(&records); err != nil {
// 		return err
// 	}

// 	//
// 	writeJSON(w, http.StatusOK, MemberDetails{})
// 	return nil
// }

// type MemberDetails struct {
// 	Registered     time.Time  `json:"registered"`
// 	EventsAttended int        `json:"eventsAttended"`
// 	RoleChange     RoleChange `json:"roleChange"`
// }

// type RoleChange struct {
// 	From string    `json:"from"`
// 	To   string    `json:"from"`
// 	Date time.Time `json:"date"`
// }
