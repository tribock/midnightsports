package server

import (
	"net/http"
	"time"
)

type Report struct {
	EventID           int        `json:"eventId"`
	Body              string     `json:"body"`
	AuthorName        string     `json:"authorName"`
	UpdatedAt         *time.Time `json:"updatedAt"`
	RegistrationCount int        `json:"registrationCount"`
	CheckinCount      int        `json:"checkinCount"`
	// Anwesende nach Geschlecht, ohne Angabe z.B. bei gelöschten Konten
	Attendance genderCount `json:"attendance"`
}

type genderCount struct {
	W     int `json:"w"`
	M     int `json:"m"`
	D     int `json:"d"`
	Other int `json:"other"`
}

// getReport liefert den Bericht eines Events. Die Teilnehmerzahlen werden immer aktuell berechnet.
func (s *Server) getReport(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	var rep Report
	err = s.db.QueryRow(r.Context(),
		`SELECT e.id, COALESCE(rp.body, ''), COALESCE(trim(a.first_name || ' ' || a.last_name), ''), rp.updated_at,
		        (SELECT count(*) FROM registrations WHERE event_id = e.id),
		        att.total, att.w, att.m, att.d, att.total - att.w - att.m - att.d
		 FROM events e
		 LEFT JOIN reports rp ON rp.event_id = e.id
		 LEFT JOIN users a ON a.id = rp.author_id
		 CROSS JOIN LATERAL (
		     SELECT count(*) AS total,
		            count(*) FILTER (WHERE u.gender = 'w') AS w,
		            count(*) FILTER (WHERE u.gender = 'm') AS m,
		            count(*) FILTER (WHERE u.gender = 'd') AS d
		     FROM checkins c LEFT JOIN users u ON u.id = c.user_id
		     WHERE c.event_id = e.id
		 ) att
		 WHERE e.id = $1`, id).
		Scan(&rep.EventID, &rep.Body, &rep.AuthorName, &rep.UpdatedAt, &rep.RegistrationCount, &rep.CheckinCount,
			&rep.Attendance.W, &rep.Attendance.M, &rep.Attendance.D, &rep.Attendance.Other)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rep)
	return nil
}

func (s *Server) saveReport(w http.ResponseWriter, r *http.Request, m *Member) error {
	id, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	var req struct{ Body string }
	if err := decode(r, &req); err != nil {
		return err
	}
	_, err = s.db.Exec(r.Context(),
		`INSERT INTO reports (event_id, body, author_id) VALUES ($1, $2, $3)
		 ON CONFLICT (event_id) DO UPDATE SET body = $2, author_id = $3, updated_at = now()`,
		id, req.Body, m.ID)
	if isForeignKeyViolation(err) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	return s.getReport(w, r, m)
}
