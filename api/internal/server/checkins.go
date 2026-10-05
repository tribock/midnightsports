package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tribock/midnightsports/api/internal/qr"
)

type checkinInput struct {
	Token          string `json:"token"`
	UserID         string `json:"userId"`
	GuestFirstName string `json:"guestFirstName"`
	GuestLastName  string `json:"guestLastName"`
}

type checkinResult struct {
	CheckinID        int       `json:"checkinId"`
	Person           *User     `json:"person,omitempty"`
	GuestName        string    `json:"guestName,omitempty"`
	Registered       bool      `json:"registered"`
	AlreadyCheckedIn bool      `json:"alreadyCheckedIn"`
	CheckedInAt      time.Time `json:"checkedInAt"`
}

type participant struct {
	CheckinID   *int       `json:"checkinId"`
	UserID      *string    `json:"userId"`
	Name        string     `json:"name"`
	BirthDate   string     `json:"birthDate"`
	Guest       bool       `json:"guest"`
	Registered  bool       `json:"registered"`
	CheckedInAt *time.Time `json:"checkedInAt"`
}

// checkIn erfasst einen Teilnehmer per QR-Token, per Benutzer-ID (Suche) oder als Gast.
func (s *Server) checkIn(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	var in checkinInput
	if err := decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()

	var cancelled bool
	if err := s.db.QueryRow(ctx, `SELECT cancelled FROM events WHERE id = $1`, eventID).Scan(&cancelled); err != nil {
		return err
	}
	if cancelled {
		return errorf(http.StatusConflict, "Der Event ist abgesagt")
	}

	res := checkinResult{}
	if in.Token == "" && in.UserID == "" {
		first, last := strings.TrimSpace(in.GuestFirstName), strings.TrimSpace(in.GuestLastName)
		if first == "" || last == "" {
			return errorf(http.StatusBadRequest, "Vorname und Name des Gasts fehlen")
		}
		err := s.db.QueryRow(ctx,
			`INSERT INTO checkins (event_id, guest_first_name, guest_last_name, checked_in_by)
			 VALUES ($1, $2, $3, $4) RETURNING id, checked_in_at`,
			eventID, first, last, u.ID).Scan(&res.CheckinID, &res.CheckedInAt)
		if err != nil {
			return err
		}
		res.GuestName = first + " " + last
		writeJSON(w, http.StatusOK, res)
		return nil
	}

	userID := in.UserID
	if in.Token != "" {
		if userID, err = qr.Verify(s.cfg.QRSigningKey, in.Token); err != nil {
			return errorf(http.StatusBadRequest, "Ungültiger QR-Code")
		}
	}
	res.Person, err = scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id::text = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return errorf(http.StatusNotFound, "Teilnehmer nicht gefunden")
	}
	if err != nil {
		return err
	}

	err = s.db.QueryRow(ctx,
		`INSERT INTO checkins (event_id, user_id, checked_in_by) VALUES ($1, $2, $3)
		 ON CONFLICT (event_id, user_id) WHERE user_id IS NOT NULL DO NOTHING
		 RETURNING id, checked_in_at`,
		eventID, res.Person.ID, u.ID).Scan(&res.CheckinID, &res.CheckedInAt)
	if errors.Is(err, pgx.ErrNoRows) {
		res.AlreadyCheckedIn = true
		err = s.db.QueryRow(ctx, `SELECT id, checked_in_at FROM checkins WHERE event_id = $1 AND user_id = $2`,
			eventID, res.Person.ID).Scan(&res.CheckinID, &res.CheckedInAt)
	}
	if err != nil {
		return err
	}
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registrations WHERE event_id = $1 AND user_id = $2)`,
		eventID, res.Person.ID).Scan(&res.Registered); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}

func (s *Server) deleteCheckIn(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	checkinID, err := pathInt(r, "checkinID")
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `DELETE FROM checkins WHERE id = $1 AND event_id = $2`, checkinID, eventID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// listParticipants liefert alle Angemeldeten und alle Eingecheckten (inkl. Gäste) eines Events.
func (s *Server) listParticipants(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT c.id, u.id::text, trim(u.first_name || ' ' || u.last_name), COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''),
		        false, true, c.checked_in_at
		 FROM registrations reg
		 JOIN users u ON u.id = reg.user_id
		 LEFT JOIN checkins c ON c.event_id = reg.event_id AND c.user_id = reg.user_id
		 WHERE reg.event_id = $1
		 UNION ALL
		 SELECT c.id, u.id::text,
		        CASE WHEN u.id IS NOT NULL THEN trim(u.first_name || ' ' || u.last_name)
		             WHEN c.guest_last_name <> '' THEN c.guest_first_name || ' ' || c.guest_last_name
		             ELSE 'Gelöschtes Konto' END,
		        COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''),
		        c.guest_last_name <> '', false, c.checked_in_at
		 FROM checkins c
		 LEFT JOIN users u ON u.id = c.user_id
		 WHERE c.event_id = $1
		   AND NOT EXISTS (SELECT 1 FROM registrations reg WHERE reg.event_id = c.event_id AND reg.user_id = c.user_id)
		 ORDER BY 3`, eventID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (participant, error) {
		var p participant
		err := row.Scan(&p.CheckinID, &p.UserID, &p.Name, &p.BirthDate, &p.Guest, &p.Registered, &p.CheckedInAt)
		return p, err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}
