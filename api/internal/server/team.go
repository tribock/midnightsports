package server

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Function ist eine Funktion im Team (z.B. JC, SC) mit Lohn pro Einsatz.
// Salary nil bedeutet: Abrechnung über die Zeiterfassung (z.B. PL), kein Betrag am Event.
type Function struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Salary *float64 `json:"salary"`
}

type TeamMember struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	Name         string  `json:"name"`
	FunctionID   *int    `json:"functionId"`
	FunctionName *string `json:"functionName"`
}

type StaffEntry struct {
	UserID       string   `json:"userId"`
	Name         string   `json:"name"`
	FunctionID   int      `json:"functionId"`
	FunctionName string   `json:"functionName"`
	Salary       *float64 `json:"salary,omitempty"`
	// PaidByTime: Abrechnung über die Zeiterfassung statt Betrag am Event
	PaidByTime bool `json:"paidByTime"`
}

// --- Funktionen (Administration) ---

func (s *Server) listFunctions(w http.ResponseWriter, r *http.Request, m *Member) error {
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, salary::float8 FROM functions WHERE project_id = $1 ORDER BY name`, m.Project.ID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Function])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

func (s *Server) createFunction(w http.ResponseWriter, r *http.Request, m *Member) error {
	return s.saveFunction(w, r, m, 0)
}

func (s *Server) updateFunction(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	return s.saveFunction(w, r, m, id)
}

// saveFunction legt eine Funktion des Projekts an (id = 0) oder aktualisiert sie.
func (s *Server) saveFunction(w http.ResponseWriter, r *http.Request, m *Member, id int) error {
	var f Function
	if err := decode(r, &f); err != nil {
		return err
	}
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		return errorf(http.StatusBadRequest, "Name fehlt")
	}
	if f.Salary != nil && *f.Salary < 0 {
		return errorf(http.StatusBadRequest, "Lohn darf nicht negativ sein")
	}

	var err error
	if id == 0 {
		err = s.db.QueryRow(r.Context(),
			`INSERT INTO functions (project_id, name, salary) VALUES ($1, $2, $3) RETURNING id`,
			m.Project.ID, f.Name, f.Salary).Scan(&f.ID)
	} else {
		err = s.db.QueryRow(r.Context(),
			`UPDATE functions SET name = $3, salary = $4 WHERE id = $1 AND project_id = $2 RETURNING id`,
			id, m.Project.ID, f.Name, f.Salary).Scan(&f.ID)
	}
	if isUniqueViolation(err) {
		return errorf(http.StatusConflict, "Eine Funktion mit diesem Namen existiert bereits")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, f)
	return nil
}

func (s *Server) deleteFunction(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	err = s.execOne(r.Context(), `DELETE FROM functions WHERE id = $1 AND project_id = $2`, id, m.Project.ID)
	if isForeignKeyViolation(err) {
		return errorf(http.StatusConflict, "Die Funktion ist in Events eingesetzt und kann nicht gelöscht werden")
	}
	if err != nil {
		return err
	}
	return noContent(w)
}

// --- Team ---

// listTeam liefert die Projektleiter, Mitarbeiter und Admins des Projekts mit ihrer Funktion.
func (s *Server) listTeam(w http.ResponseWriter, r *http.Request, m *Member) error {
	rows, err := s.db.Query(r.Context(),
		`SELECT u.id::text, COALESCE(u.email, ''), pm.role, trim(u.first_name || ' ' || u.last_name), f.id, f.name
		 FROM project_members pm
		 JOIN users u ON u.id = pm.user_id
		 LEFT JOIN functions f ON f.id = pm.function_id
		 WHERE pm.project_id = $1 AND pm.role IN ('projektleiter', 'mitarbeiter', 'admin')
		 ORDER BY u.last_name, u.first_name, u.email`, m.Project.ID)
	if err != nil {
		return err
	}
	team, err := pgx.CollectRows(rows, pgx.RowToStructByPos[TeamMember])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, team)
	return nil
}

// setMemberFunction legt die Funktion einer Person im Team fest (null = keine). Die Funktion bestimmt den Lohn,
// deshalb wird die Änderung protokolliert.
func (s *Server) setMemberFunction(w http.ResponseWriter, r *http.Request, m *Member) error {
	var req struct {
		FunctionID *int `json:"functionId"`
	}
	if err := decode(r, &req); err != nil {
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
	var name *string
	err = tx.QueryRow(ctx,
		`UPDATE project_members pm SET function_id = $3
		 WHERE pm.project_id = $1 AND pm.user_id = $2 AND pm.role IN ('projektleiter', 'mitarbeiter', 'admin')
		 RETURNING (SELECT f.name FROM functions f WHERE f.id = $3)`,
		m.Project.ID, target.ID, req.FunctionID).Scan(&name)
	switch {
	case isForeignKeyViolation(err):
		return errorf(http.StatusBadRequest, "Unbekannte Funktion")
	case errors.Is(err, pgx.ErrNoRows):
		return errorf(http.StatusNotFound, "Person ist nicht im Team")
	case err != nil:
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionFunctionChanged,
		Target: target, Details: map[string]any{"funktion": name}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return noContent(w)
}

// --- Einsatz am Event ---

// listStaff liefert das Team eines Events. Den Lohn sehen nur Projektleiter und Admins.
func (s *Server) listStaff(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT es.user_id::text, trim(u.first_name || ' ' || u.last_name), f.id, f.name, es.salary::float8,
		        es.salary IS NULL
		 FROM event_staff es
		 JOIN users u ON u.id = es.user_id
		 JOIN functions f ON f.id = es.function_id
		 WHERE es.event_id = $1
		 ORDER BY f.name, u.last_name, u.first_name`, eventID)
	if err != nil {
		return err
	}
	staff, err := pgx.CollectRows(rows, pgx.RowToStructByPos[StaffEntry])
	if err != nil {
		return err
	}
	if !isProjectLead(m) {
		for i := range staff {
			staff[i].Salary = nil
		}
	}
	writeJSON(w, http.StatusOK, staff)
	return nil
}

type staffInput struct {
	UserID string `json:"userId"`
}

type staffSnapshot struct {
	functionID int
	salary     *float64
	signature  []byte
	signedAt   *time.Time
	signedBy   *string
}

// saveStaff ersetzt den Einsatz eines Events. Funktion und Betrag kommen aus dem Team bzw. der Funktion
// und werden gespeichert. Wer mit derselben Funktion eingeteilt bleibt, behält seinen bisherigen Betrag,
// damit spätere Lohnänderungen vergangene Events nicht verändern, und eine bereits erfasste Unterschrift (Auszahlung).
// Funktionen ohne festen Lohn werden über die Zeiterfassung abgerechnet und erhalten keinen Betrag.
func saveStaff(ctx context.Context, tx pgx.Tx, projectID, eventID int, staff []staffInput) error {
	// Bisherige Einträge merken: wer mit derselben Funktion eingeteilt bleibt, behält Betrag und Unterschrift
	existing, err := loadStaffSnapshots(ctx, tx, eventID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_staff WHERE event_id = $1`, eventID); err != nil {
		return err
	}

	ids := make([]string, 0, len(staff))
	for _, in := range staff {
		id := strings.ToLower(in.UserID)
		if !validUUID(id) {
			return errorf(http.StatusBadRequest, "Person ist nicht im Team")
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	members, err := loadTeamMembers(ctx, tx, projectID, ids)
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, id := range ids {
		m, ok := members[id]
		if !ok {
			return errorf(http.StatusBadRequest, "Person ist nicht im Team")
		}
		if m.functionID == nil {
			return errorf(http.StatusBadRequest, m.name+" hat keine Funktion im Team")
		}
		prev, hadPrev := existing[id]
		snap := staffEntry(prev, hadPrev, *m.functionID, m.salary)
		batch.Queue(`INSERT INTO event_staff (event_id, user_id, function_id, salary, signature, signed_at, signed_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			eventID, id, snap.functionID, snap.salary, snap.signature, snap.signedAt, snap.signedBy)
	}
	return tx.SendBatch(ctx, batch).Close()
}

type teamMember struct {
	name       string
	functionID *int
	salary     *float64
}

// loadTeamMembers lädt Funktion und Lohn der gewählten Personen in einer Abfrage. Nur Personen im Team des Projekts.
func loadTeamMembers(ctx context.Context, tx pgx.Tx, projectID int, ids []string) (map[string]teamMember, error) {
	rows, err := tx.Query(ctx,
		`SELECT u.id::text, trim(u.first_name || ' ' || u.last_name), pm.function_id, f.salary::float8
		 FROM project_members pm
		 JOIN users u ON u.id = pm.user_id
		 LEFT JOIN functions f ON f.id = pm.function_id
		 WHERE pm.project_id = $1 AND u.id = ANY($2::uuid[]) AND pm.role IN ('projektleiter', 'mitarbeiter', 'admin')`,
		projectID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := map[string]teamMember{}
	for rows.Next() {
		var id string
		var m teamMember
		if err := rows.Scan(&id, &m.name, &m.functionID, &m.salary); err != nil {
			return nil, err
		}
		members[id] = m
	}
	return members, rows.Err()
}

// loadStaffSnapshots lädt den bisherigen Einsatz eines Events pro Person.
func loadStaffSnapshots(ctx context.Context, tx pgx.Tx, eventID int) (map[string]staffSnapshot, error) {
	rows, err := tx.Query(ctx,
		`SELECT user_id::text, function_id, salary::float8, signature, signed_at, signed_by::text
		 FROM event_staff WHERE event_id = $1`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	existing := map[string]staffSnapshot{}
	for rows.Next() {
		var id string
		var snap staffSnapshot
		if err := rows.Scan(&id, &snap.functionID, &snap.salary, &snap.signature, &snap.signedAt, &snap.signedBy); err != nil {
			return nil, err
		}
		existing[id] = snap
	}
	return existing, rows.Err()
}

// staffEntry bestimmt Funktion, Betrag und Unterschrift einer eingeteilten Person.
// Der Betrag kommt aus der Funktion und ist nicht editierbar, ohne festen Lohn wird über die Zeiterfassung
// abgerechnet. War die Person bereits mit derselben Funktion und einem Betrag eingeteilt, bleibt der
// bisherige Eintrag (Betrag und Unterschrift) erhalten.
func staffEntry(prev staffSnapshot, hadPrev bool, functionID int, salary *float64) staffSnapshot {
	if hadPrev && salary != nil && prev.functionID == functionID && prev.salary != nil {
		return prev
	}
	return staffSnapshot{functionID: functionID, salary: salary}
}
