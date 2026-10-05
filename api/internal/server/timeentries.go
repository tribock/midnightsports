package server

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type TimeEntry struct {
	ID          int     `json:"id"`
	UserID      string  `json:"userId"`
	UserName    string  `json:"userName"`
	Date        string  `json:"date"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
	EventID     *int    `json:"eventId"`
	EventDate   string  `json:"eventDate"`
}

const timeEntryQuery = `SELECT t.id, t.user_id::text, trim(u.first_name || ' ' || u.last_name),
	to_char(t.date, 'YYYY-MM-DD'), t.hours::float8, t.description, t.event_id,
	COALESCE(to_char(e.starts_at AT TIME ZONE 'Europe/Zurich', 'YYYY-MM-DD'), '')
	FROM time_entries t
	JOIN users u ON u.id = t.user_id
	LEFT JOIN events e ON e.id = t.event_id`

func (s *Server) queryTimeEntries(r *http.Request, u *User) ([]TimeEntry, error) {
	// Admins sehen mit ?all=true alle Einträge, sonst nur die eigenen
	all := u.Role == RoleAdmin && r.URL.Query().Get("all") == "true"
	rows, err := s.db.Query(r.Context(),
		timeEntryQuery+` WHERE $1 OR t.user_id = $2 ORDER BY t.date DESC, t.id DESC`, all, u.ID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[TimeEntry])
}

func (s *Server) listTimeEntries(w http.ResponseWriter, r *http.Request, u *User) error {
	entries, err := s.queryTimeEntries(r, u)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, entries)
	return nil
}

func (s *Server) createTimeEntry(w http.ResponseWriter, r *http.Request, u *User) error {
	var in TimeEntry
	if err := decode(r, &in); err != nil {
		return err
	}
	date, err := time.Parse(time.DateOnly, in.Date)
	if err != nil {
		return errorf(http.StatusBadRequest, "Ungültiges Datum")
	}
	if in.Hours <= 0 || in.Hours > 24 {
		return errorf(http.StatusBadRequest, "Dauer muss zwischen 0 und 24 Stunden liegen")
	}
	var id int
	err = s.db.QueryRow(r.Context(),
		`INSERT INTO time_entries (user_id, date, hours, description, event_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		u.ID, date, in.Hours, strings.TrimSpace(in.Description), in.EventID).Scan(&id)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), timeEntryQuery+` WHERE t.id = $1`, id)
	if err != nil {
		return err
	}
	entry, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[TimeEntry])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, entry)
	return nil
}

func (s *Server) deleteTimeEntry(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(),
		`DELETE FROM time_entries WHERE id = $1 AND (user_id = $2 OR $3)`, id, u.ID, u.Role == RoleAdmin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// exportTimeEntries liefert alle Zeiteinträge als CSV (Semikolon, UTF-8 mit BOM für Excel).
func (s *Server) exportTimeEntries(w http.ResponseWriter, r *http.Request, u *User) error {
	q := r.URL.Query()
	q.Set("all", "true")
	r.URL.RawQuery = q.Encode()
	entries, err := s.queryTimeEntries(r, u)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="zeiterfassung.csv"`)
	w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.Write([]string{"Datum", "Name", "Stunden", "Beschreibung", "Event"})
	for _, e := range entries {
		cw.Write([]string{e.Date, e.UserName, strconv.FormatFloat(e.Hours, 'f', 2, 64), e.Description, e.EventDate})
	}
	cw.Flush()
	return cw.Error()
}
