package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type Event struct {
	ID                int       `json:"id"`
	Date              string    `json:"date"`
	StartTime         string    `json:"startTime"`
	EndTime           string    `json:"endTime"`
	StartsAt          time.Time `json:"startsAt"`
	EndsAt            time.Time `json:"endsAt"`
	Cancelled         bool      `json:"cancelled"`
	Location          Location  `json:"location"`
	Leaders           []Person  `json:"leaders"`
	Registered        bool      `json:"registered"`
	RegistrationCount int       `json:"registrationCount"`
	CheckinCount      int       `json:"checkinCount"`
	RegistrationOpen  bool      `json:"registrationOpen"`
}

type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type eventInput struct {
	Date       string   `json:"date"`
	StartTime  string   `json:"startTime"`
	EndTime    string   `json:"endTime"`
	LocationID int      `json:"locationId"`
	LeaderIDs  []string `json:"leaderIds"`
}

const eventQuery = `SELECT e.id, e.starts_at, e.ends_at, e.cancelled,
	l.id, l.name, l.street, l.zip, l.city, l.is_default,
	EXISTS (SELECT 1 FROM registrations r WHERE r.event_id = e.id AND r.user_id = $1),
	(SELECT count(*) FROM registrations r WHERE r.event_id = e.id),
	(SELECT count(*) FROM checkins c WHERE c.event_id = e.id),
	COALESCE((SELECT json_agg(json_build_object('id', u.id, 'name', trim(u.first_name || ' ' || u.last_name))
	          ORDER BY u.last_name, u.first_name)
	          FROM event_leaders el JOIN users u ON u.id = el.user_id WHERE el.event_id = e.id), '[]')
	FROM events e JOIN locations l ON l.id = e.location_id`

func (s *Server) scanEvent(row pgx.Row) (*Event, error) {
	var e Event
	l := &e.Location
	err := row.Scan(&e.ID, &e.StartsAt, &e.EndsAt, &e.Cancelled,
		&l.ID, &l.Name, &l.Street, &l.Zip, &l.City, &l.IsDefault,
		&e.Registered, &e.RegistrationCount, &e.CheckinCount, &e.Leaders)
	if err != nil {
		return nil, err
	}
	start, end := e.StartsAt.In(s.tz), e.EndsAt.In(s.tz)
	e.Date = start.Format(time.DateOnly)
	e.StartTime = start.Format("15:04")
	e.EndTime = end.Format("15:04")
	e.RegistrationOpen = !e.Cancelled && time.Now().Before(e.StartsAt)
	return &e, nil
}

// eventTimes berechnet Start und Ende. Liegt das Ende vor dem Start, endet der Event am Folgetag.
func eventTimes(tz *time.Location, date, startTime, endTime string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02 15:04", date+" "+startTime, tz)
	if err != nil {
		return time.Time{}, time.Time{}, errorf(http.StatusBadRequest, "Ungültiges Datum oder ungültige Startzeit")
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", date+" "+endTime, tz)
	if err != nil {
		return time.Time{}, time.Time{}, errorf(http.StatusBadRequest, "Ungültige Endzeit")
	}
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return start, end, nil
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request, u *User) error {
	query := eventQuery + ` WHERE e.ends_at > now() ORDER BY e.starts_at`
	if r.URL.Query().Get("past") == "true" {
		query = eventQuery + ` WHERE e.ends_at <= now() ORDER BY e.starts_at DESC LIMIT 100`
	}
	rows, err := s.db.Query(r.Context(), query, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	events := []*Event{}
	for rows.Next() {
		e, err := s.scanEvent(rows)
		if err != nil {
			return err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, events)
	return nil
}

func (s *Server) loadEvent(ctx context.Context, id int, userID string) (*Event, error) {
	return s.scanEvent(s.db.QueryRow(ctx, eventQuery+` WHERE e.id = $2`, userID, id))
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	e, err := s.loadEvent(r.Context(), id, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request, u *User) error {
	return s.saveEvent(w, r, u, 0)
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	return s.saveEvent(w, r, u, id)
}

// saveEvent legt einen Event an (id = 0) oder aktualisiert ihn.
func (s *Server) saveEvent(w http.ResponseWriter, r *http.Request, u *User, id int) error {
	var in eventInput
	if err := decode(r, &in); err != nil {
		return err
	}
	start, end, err := eventTimes(s.tz, in.Date, in.StartTime, in.EndTime)
	if err != nil {
		return err
	}
	if len(in.LeaderIDs) == 0 {
		in.LeaderIDs = []string{u.ID}
	}

	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if in.LocationID == 0 {
		if err := tx.QueryRow(ctx, `SELECT id FROM locations ORDER BY is_default DESC, id LIMIT 1`).Scan(&in.LocationID); err != nil {
			return err
		}
	}
	var validLeaders int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE id::text = ANY($1) AND role IN ('projektleiter', 'admin')`,
		in.LeaderIDs).Scan(&validLeaders); err != nil {
		return err
	}
	if validLeaders != len(in.LeaderIDs) {
		return errorf(http.StatusBadRequest, "Nur Projektleiter können einem Event zugewiesen werden")
	}

	if id == 0 {
		err = tx.QueryRow(ctx,
			`INSERT INTO events (starts_at, ends_at, location_id, created_by) VALUES ($1, $2, $3, $4) RETURNING id`,
			start, end, in.LocationID, u.ID).Scan(&id)
	} else {
		err = tx.QueryRow(ctx,
			`UPDATE events SET starts_at = $2, ends_at = $3, location_id = $4 WHERE id = $1 RETURNING id`,
			id, start, end, in.LocationID).Scan(&id)
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_leaders WHERE event_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO event_leaders (event_id, user_id) SELECT $1, unnest($2::uuid[])`, id, in.LeaderIDs); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	e, err := s.loadEvent(ctx, id, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}

func (s *Server) cancelEvent(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	var req struct{ Cancelled bool }
	if err := decode(r, &req); err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE events SET cancelled = $2 WHERE id = $1`, id, req.Cancelled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	e, err := s.loadEvent(r.Context(), id, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}

func (s *Server) register(w http.ResponseWriter, r *http.Request, u *User) error {
	if !u.ProfileComplete {
		return errIncomplete
	}
	return s.changeRegistration(w, r, u, `INSERT INTO registrations (event_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`)
}

func (s *Server) unregister(w http.ResponseWriter, r *http.Request, u *User) error {
	return s.changeRegistration(w, r, u, `DELETE FROM registrations WHERE event_id = $1 AND user_id = $2`)
}

// changeRegistration meldet an oder ab. Beides ist nur bis zum Start des Events möglich.
func (s *Server) changeRegistration(w http.ResponseWriter, r *http.Request, u *User, sql string) error {
	id, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	e, err := s.loadEvent(r.Context(), id, u.ID)
	if err != nil {
		return err
	}
	if !e.RegistrationOpen {
		return errorf(http.StatusConflict, "An- und Abmeldung ist für diesen Event nicht mehr möglich")
	}
	if _, err := s.db.Exec(r.Context(), sql, id, u.ID); err != nil {
		return err
	}
	e, err = s.loadEvent(r.Context(), id, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, e)
	return nil
}
