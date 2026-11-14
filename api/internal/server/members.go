package server

import (
	"context"
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

// setMemberRole ändert die Rolle eines Mitarbeiters (Projekt-Admins). Rollen haben nur Mitarbeiter (Status);
// Teilnehmer wird man, indem der Status Mitarbeiter entzogen wird.
func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct{ Role string }
	if err := decode(r, &req); err != nil {
		return err
	}
	if !slices.Contains(allRoles, req.Role) || req.Role == RoleTeilnehmer {
		return errorf(http.StatusBadRequest, "Ungültige Rolle")
	}
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	if target.ID == m.ID && req.Role != RoleAdmin {
		return errorf(http.StatusBadRequest, "Die eigene Admin-Rolle kann nicht entzogen werden")
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old string
	var status UserStatus
	if err := tx.QueryRow(ctx,
		`SELECT role, status FROM project_members WHERE project_id = $1 AND user_id = $2 FOR UPDATE`,
		m.Project.ID, target.ID).Scan(&old, &status); err != nil {
		return err
	}
	if status != StatusMitarbeiter {
		return errorf(http.StatusBadRequest, "Rollen haben nur Mitarbeiter: zuerst den Status Mitarbeiter vergeben")
	}
	if old == req.Role {
		return noContent(w)
	}
	if old == RoleAdmin {
		if err := keepLastAdmin(ctx, tx, m.Project.ID, target.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE project_members SET role = $3 WHERE project_id = $1 AND user_id = $2`,
		m.Project.ID, target.ID, req.Role); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionRoleChanged,
		Target: target, Details: map[string]any{"vorher": old, "nachher": req.Role}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// setMemberStatus ändert den Status eines Mitglieds (Projekt-Admins). Wer den Status Mitarbeiter bekommt, wird
// Mitarbeiter mit Funktion (functionId); wer ihn verliert, wird Teilnehmer ohne Funktion.
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
	role, functionID, err := staffMembership(req.Status, req.FunctionID)
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
	var oldRole string
	if err := tx.QueryRow(ctx,
		`SELECT status, role FROM project_members WHERE project_id = $1 AND user_id = $2 FOR UPDATE`,
		m.Project.ID, target.ID).Scan(&old, &oldRole); err != nil {
		return err
	}
	// Bleibt jemand Mitarbeiter, ändern sich Rolle und Funktion nur über das Team
	if old == req.Status {
		return noContent(w)
	}
	if target.ID == m.ID && old == StatusMitarbeiter {
		return errorf(http.StatusBadRequest, "Den eigenen Status Mitarbeiter kann man nicht entziehen")
	}
	var function *string
	err = tx.QueryRow(ctx,
		`UPDATE project_members SET status = $3, role = $4, function_id = $5 WHERE project_id = $1 AND user_id = $2
		 RETURNING (SELECT f.name FROM functions f WHERE f.id = $5)`,
		m.Project.ID, target.ID, req.Status, role, functionID).Scan(&function)
	if isForeignKeyViolation(err) {
		return errorf(http.StatusBadRequest, "Unbekannte Funktion")
	}
	if err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionStatusChanged,
		Target: target, Details: map[string]any{"vorher": old, "nachher": req.Status, "funktion": function,
			"rolle_vorher": oldRole, "rolle": role}}); err != nil {
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

// MemberDetails: Verlauf eines Mitarbeiters im Projekt
type MemberDetails struct {
	Registered      time.Time        `json:"registered"` // Konto erstellt
	Joined          time.Time        `json:"joined"`     // Mitglied im Projekt seit
	EventsWorked    int              `json:"eventsWorked"`
	EventsPlanned   int              `json:"eventsPlanned"`
	EventsAttended  int              `json:"eventsAttended"`
	Events          []MemberEvent    `json:"events"`
	FunctionChanges []FunctionChange `json:"functionChanges"`
}

// MemberEvent: Event wie in der Eventliste, dazu die Funktion der Person an diesem Event
type MemberEvent struct {
	*Event
	Function string `json:"function"`
}

type FunctionChange struct {
	At   time.Time `json:"at"`
	By   string    `json:"by"`
	From *string   `json:"from"` // nil: vorher kein Mitarbeiter
	To   *string   `json:"to"`   // nil: Status Mitarbeiter entzogen
}

// getMemberDetails liefert den Verlauf eines Mitarbeiters: Registrierung, Einsätze und Änderungen der Funktion.
func (s *Server) getMemberDetails(w http.ResponseWriter, r *http.Request, m *Member) error {
	target, err := s.projectMemberUser(r, m)
	if err != nil {
		return err
	}
	ctx := r.Context()
	var d MemberDetails
	// Nur Mitarbeiter haben einen Verlauf, alle anderen ergeben 404
	if err := s.db.QueryRow(ctx,
		`SELECT u.created_at, pm.joined_at,
		        (SELECT count(*) FROM checkins c JOIN events e ON e.id = c.event_id
		         WHERE e.project_id = pm.project_id AND c.user_id = pm.user_id)
		 FROM project_members pm
		 JOIN users u ON u.id = pm.user_id
		 WHERE pm.project_id = $1 AND pm.user_id = $2 AND pm.status = 'mitarbeiter'`,
		m.Project.ID, target.ID).Scan(&d.Registered, &d.Joined, &d.EventsAttended); err != nil {
		return err
	}

	if d.Events, err = s.memberEvents(ctx, m, target.ID); err != nil {
		return err
	}
	now := time.Now()
	for _, e := range d.Events {
		if !e.Cancelled {
			if e.StartsAt.Before(now) {
				d.EventsWorked++
			} else {
				d.EventsPlanned++
			}
		}
	}

	// Funktion ändert sich über das Team oder zusammen mit dem Status Mitarbeiter
	rows, err := s.db.Query(ctx,
		`SELECT at, actor_name, NULL::text, details->>'funktion'
		 FROM audit_log
		 WHERE project_id = $1 AND target_user_id = $2
		   AND (action = $3 OR (action = $4 AND 'mitarbeiter' IN (details->>'vorher', details->>'nachher')))
		 ORDER BY id`, m.Project.ID, target.ID, actionFunctionChanged, actionStatusChanged)
	if err != nil {
		return err
	}
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[FunctionChange]) // From folgt in functionChanges
	if err != nil {
		return err
	}
	d.FunctionChanges = functionChanges(entries)

	writeJSON(w, http.StatusOK, d)
	return nil
}

// memberEvents lädt die Events des Projekts, an denen die Person eingeteilt ist, neuste zuerst. Die Events kommen
// aus eventQuery wie in der Eventliste, die Funktion am Event aus einer zweiten Abfrage.
func (s *Server) memberEvents(ctx context.Context, m *Member, userID string) ([]MemberEvent, error) {
	functions := map[int]string{}
	rows, err := s.db.Query(ctx,
		`SELECT es.event_id, f.name
		 FROM event_staff es
		 JOIN events e ON e.id = es.event_id
		 JOIN functions f ON f.id = es.function_id
		 WHERE e.project_id = $1 AND es.user_id = $2`, m.Project.ID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		functions[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.Query(ctx, eventQuery+`
		 WHERE e.project_id = $2 AND e.id IN (SELECT event_id FROM event_staff WHERE user_id = $3)
		 ORDER BY e.starts_at DESC, e.id`, m.ID, m.Project.ID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []MemberEvent{}
	for rows.Next() {
		e, err := s.scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, MemberEvent{e, functions[e.ID]})
	}
	return events, rows.Err()
}

// functionChanges ergänzt die bisherige Funktion (From) aus dem vorherigen Eintrag, weil das Audit nur die neue
// speichert. Erwartet die Einträge chronologisch.
func functionChanges(entries []FunctionChange) []FunctionChange {
	var prev *string
	for i := range entries {
		entries[i].From = prev
		prev = entries[i].To
	}
	return entries
}
