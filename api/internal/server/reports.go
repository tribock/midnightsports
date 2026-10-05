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
}

// getReport liefert den Bericht eines Events. Die Teilnehmerzahlen werden immer aktuell berechnet.
func (s *Server) getReport(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	var rep Report
	err = s.db.QueryRow(r.Context(),
		`SELECT e.id, COALESCE(rp.body, ''), COALESCE(trim(a.first_name || ' ' || a.last_name), ''), rp.updated_at,
		        (SELECT count(*) FROM registrations WHERE event_id = e.id),
		        (SELECT count(*) FROM checkins WHERE event_id = e.id)
		 FROM events e
		 LEFT JOIN reports rp ON rp.event_id = e.id
		 LEFT JOIN users a ON a.id = rp.author_id
		 WHERE e.id = $1`, id).
		Scan(&rep.EventID, &rep.Body, &rep.AuthorName, &rep.UpdatedAt, &rep.RegistrationCount, &rep.CheckinCount)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rep)
	return nil
}

func (s *Server) saveReport(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
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
		id, req.Body, u.ID)
	if err != nil {
		return err
	}
	return s.getReport(w, r, u)
}
