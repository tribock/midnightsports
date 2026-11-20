package server

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// Statistics fasst die vergangenen, nicht abgesagten Events eines Zeitraums zusammen.
type Statistics struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	EventCount int             `json:"eventCount"`
	Attendance StatsAttendance `json:"attendance"`
	Team       []StatsMember   `json:"team"`
}

// StatsAttendance: Check-ins total und nach Geschlecht, Other = ohne Angabe oder gelöschtes Konto
type StatsAttendance struct {
	Total int `json:"total"`
	W     int `json:"w"`
	M     int `json:"m"`
	D     int `json:"d"`
	Other int `json:"other"`
}

// StatsMember: Einsätze einer Person im Zeitraum. Paid zählt nur unterschriebene Beträge (Auszahlung),
// Hours die Zeiterfassung (Funktionen ohne festen Lohn wie PL).
type StatsMember struct {
	UserID   string  `json:"userId"`
	Name     string  `json:"name"`
	Function string  `json:"function"`
	Events   int     `json:"events"`
	Paid     float64 `json:"paid"`
	Hours    float64 `json:"hours"`
}

// statsRange prüft den Zeitraum (beide Tage inklusive) und liefert Beginn und Ende in der Zeitzone des Projekts.
func statsRange(tz *time.Location, from, to string) (time.Time, time.Time, error) {
	start, err1 := time.ParseInLocation(time.DateOnly, from, tz)
	end, err2 := time.ParseInLocation(time.DateOnly, to, tz)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, errorf(http.StatusBadRequest, "Ungültiger Zeitraum")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, errorf(http.StatusBadRequest, "Das Ende liegt vor dem Beginn")
	}
	return start, end.AddDate(0, 0, 1), nil
}

// Vergangene, nicht abgesagte Events des Projekts im Zeitraum ($1 Projekt, $2 Beginn, $3 Ende exklusiv)
const statsEvents = `ev AS (SELECT id FROM events
	WHERE project_id = $1 AND NOT cancelled AND starts_at >= $2 AND starts_at < LEAST($3, now()))`

// getStatistics liefert Teilnehmende nach Geschlecht, Einsätze und Auszahlungen im Zeitraum ?from=…&to=… (Projektleiter).
func (s *Server) getStatistics(w http.ResponseWriter, r *http.Request, m *Member) error {
	q := r.URL.Query()
	start, end, err := statsRange(s.tz, q.Get("from"), q.Get("to"))
	if err != nil {
		return err
	}
	ctx := r.Context()
	st := Statistics{From: q.Get("from"), To: q.Get("to")}
	a := &st.Attendance
	if err := s.db.QueryRow(ctx,
		`WITH `+statsEvents+`
		 SELECT (SELECT count(*) FROM ev), count(c.id),
		        count(*) FILTER (WHERE u.gender = 'w'),
		        count(*) FILTER (WHERE u.gender = 'm'),
		        count(*) FILTER (WHERE u.gender = 'd')
		 FROM ev
		 JOIN checkins c ON c.event_id = ev.id
		 LEFT JOIN users u ON u.id = c.user_id`, m.Project.ID, start, end).
		Scan(&st.EventCount, &a.Total, &a.W, &a.M, &a.D); err != nil {
		return err
	}
	a.Other = a.Total - a.W - a.M - a.D

	// Team: Einsätze an den Events und Stunden der Zeiterfassung, auch wer nur eines von beiden hat
	rows, err := s.db.Query(ctx,
		`WITH `+statsEvents+`,
		 work AS (SELECT es.user_id, count(*) AS events,
		                 COALESCE(sum(es.salary) FILTER (WHERE es.signed_at IS NOT NULL), 0) AS paid
		          FROM event_staff es JOIN ev ON ev.id = es.event_id
		          GROUP BY es.user_id),
		 hours AS (SELECT user_id, sum(hours) AS hours FROM time_entries
		           WHERE project_id = $1 AND date >= $4 AND date <= $5
		           GROUP BY user_id)
		 SELECT u.id::text, trim(u.first_name || ' ' || u.last_name), COALESCE(f.name, ''),
		        COALESCE(w.events, 0), COALESCE(w.paid, 0)::float8, COALESCE(h.hours, 0)::float8
		 FROM work w
		 FULL JOIN hours h ON h.user_id = w.user_id
		 JOIN users u ON u.id = COALESCE(w.user_id, h.user_id)
		 LEFT JOIN project_members pm ON pm.project_id = $1 AND pm.user_id = u.id
		 LEFT JOIN functions f ON f.id = pm.function_id
		 ORDER BY u.last_name, u.first_name, u.id`,
		m.Project.ID, start, end, q.Get("from"), q.Get("to"))
	if err != nil {
		return err
	}
	if st.Team, err = pgx.CollectRows(rows, pgx.RowToStructByPos[StatsMember]); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, st)
	return nil
}
