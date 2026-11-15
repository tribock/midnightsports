package server

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ExpenseEntry struct {
	ID        int     `json:"id"`
	UserID    string  `json:"userId"`
	UserName  string  `json:"userName"`
	Date      string  `json:"date"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
	EventID   *int    `json:"eventId"`
	EventDate string  `json:"eventDate"`
}

const expenseEntryQuery = `SELECT x.id, x.user_id::text, trim(u.first_name || ' ' || u.last_name),
	to_char(x.date, 'YYYY-MM-DD'), x.amount::float8, x.reason, x.event_id,
	COALESCE(to_char(e.starts_at AT TIME ZONE 'Europe/Zurich', 'YYYY-MM-DD'), '')
	FROM expenses x
	JOIN users u ON u.id = x.user_id
	LEFT JOIN events e ON e.id = x.event_id`

// queryExpenseEntries liefert die eigenen Einträge im Projekt, mit all = true die aller Personen.
func (s *Server) queryExpenseEntries(ctx context.Context, m *Member, all bool) ([]ExpenseEntry, error) {
	rows, err := s.db.Query(ctx,
		expenseEntryQuery+` WHERE x.project_id = $3 AND ($1 OR x.user_id = $2) ORDER BY x.date DESC, x.id DESC`,
		all, m.ID, m.Project.ID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[ExpenseEntry])
}

func (s *Server) listExpenseEntries(w http.ResponseWriter, r *http.Request, m *Member) error {
	// Projektleiter und Admins sehen mit ?all=true alle Einträge, sonst nur die eigenen
	all := slices.Contains(projectLeads, m.Role) && r.URL.Query().Get("all") == "true"
	entries, err := s.queryExpenseEntries(r.Context(), m, all)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, entries)
	return nil
}

func (s *Server) createExpenseEntry(w http.ResponseWriter, r *http.Request, m *Member) error {
	var in ExpenseEntry
	if err := decode(r, &in); err != nil {
		return err
	}
	date, err := time.Parse(time.DateOnly, in.Date)
	if err != nil {
		return errorf(http.StatusBadRequest, "Ungültiges Datum")
	}
	if in.Amount <= 0 || in.Amount > 1000 {
		fmt.Println(in.Amount)
		return errorf(http.StatusBadRequest, "Betrag muss zwischen 0 und 1000 Franken liegen")
	}
	// Ein Event muss zum Projekt gehören
	var id int
	err = s.db.QueryRow(r.Context(),
		`INSERT INTO expenses (project_id, user_id, date, amount, reason, event_id)
		 SELECT $1, $2, $3, $4, $5, $6
		 WHERE $6::int IS NULL OR EXISTS (SELECT 1 FROM events WHERE id = $6 AND project_id = $1)
		 RETURNING id`,
		m.Project.ID, m.ID, date, in.Amount, strings.TrimSpace(in.Reason), in.EventID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) || isForeignKeyViolation(err) {
		return errorf(http.StatusBadRequest, "Unbekannter Event")
	}
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), expenseEntryQuery+` WHERE x.id = $1`, id)
	if err != nil {
		return err
	}
	entry, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[ExpenseEntry])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, entry)
	return nil
}

func (s *Server) deleteExpenseEntry(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	if err := s.execOne(r.Context(),
		`DELETE FROM expenses WHERE id = $1 AND project_id = $2 AND (user_id = $3 OR $4)`,
		id, m.Project.ID, m.ID, slices.Contains(projectLeads, m.Role)); err != nil {
		return err
	}
	return noContent(w)
}

// exportExpenseEntries liefert alle Spesen des Projekts als CSV (Semikolon, UTF-8 mit BOM für Excel).
// Ein Export von Personendaten wird protokolliert.
func (s *Server) exportExpenseEntries(w http.ResponseWriter, r *http.Request, m *Member) error {
	entries, err := s.queryExpenseEntries(r.Context(), m, true)
	if err != nil {
		return err
	}
	if err := s.audit(r.Context(), s.db, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionExport,
		Details: map[string]any{"daten": "spesen", "eintraege": len(entries)}}); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="spesen.csv"`)
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.Write([]string{"Datum", "Name", "Betrag", "Begründung", "Event"})
	for _, e := range entries {
		cw.Write([]string{e.Date, csvText(e.UserName), strconv.FormatFloat(e.Amount, 'f', 2, 64), csvText(e.Reason), e.EventDate})
	}
	cw.Flush()
	return cw.Error()
}
