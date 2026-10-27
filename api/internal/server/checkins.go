package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tribock/midnightsports/api/internal/qr"
)

type checkinInput struct {
	Token  string `json:"token"`
	UserID string `json:"userId"`
}

type checkinResult struct {
	CheckinID        int        `json:"checkinId"`
	Person           *User      `json:"person"`
	Status           UserStatus `json:"status"`
	Registered       bool       `json:"registered"`
	AlreadyCheckedIn bool       `json:"alreadyCheckedIn"`
	CheckedInAt      *time.Time `json:"checkedInAt"`
	// Sanctions: geltende Verwarnungen und Sperren, auf dem scannenden Gerät als Meldung angezeigt
	Sanctions []Sanction `json:"sanctions"`
	// Blocked: gesperrt, nicht eingecheckt
	Blocked bool `json:"blocked"`
}

type participant struct {
	CheckinID   *int       `json:"checkinId"`
	UserID      *string    `json:"userId"`
	Name        string     `json:"name"`
	BirthDate   string     `json:"birthDate"`
	Status      UserStatus `json:"status"`
	Guest       bool       `json:"guest"`
	Registered  bool       `json:"registered"`
	CheckedInAt *time.Time `json:"checkedInAt"`
	Sanctions   []Sanction `json:"sanctions"`
}

// checkIn erfasst ein Mitglied per QR-Token oder per Benutzer-ID (Suche).
// Gäste ohne Konto werden über createGuest erfasst.
func (s *Server) checkIn(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	var in checkinInput
	if err := decode(r, &in); err != nil {
		return err
	}
	userID := in.UserID
	if in.Token != "" {
		if userID, err = qr.Verify(s.cfg.QRSigningKey, in.Token); err != nil {
			return errorf(http.StatusBadRequest, "Ungültiger QR-Code")
		}
	}
	if userID == "" {
		return errorf(http.StatusBadRequest, "QR-Code oder Teilnehmer fehlt")
	}
	if err := s.requireOpenEvent(r.Context(), eventID); err != nil {
		return err
	}
	res, err := s.checkInUser(r.Context(), m, eventID, userID)
	if err != nil {
		return err
	}
	res.Person = res.Person.visibleTo(m.ID, m.EmergencyAccess)
	writeJSON(w, http.StatusOK, res)
	return nil
}

// createGuest erfasst einen Gast als Person ohne Login, meldet ihn beim Projekt an und checkt ihn ein.
// Es gelten dieselben Pflichtfelder wie im Profil, nur die Email ist freiwillig.
func (s *Server) createGuest(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	var p struct {
		User
		Status UserStatus `json:"status"`
	}
	if err := decode(r, &p); err != nil {
		return err
	}
	if !p.Status.Valid() {
		return errorf(http.StatusBadRequest, "Bitte Status auswählen")
	}
	if err := checkStatusChange(m.Role, "", p.Status); err != nil {
		return err
	}
	birth, err := validateProfile(&p.User)
	if err != nil {
		return err
	}
	email, err := optionalEmail(p.Email)
	if err != nil {
		return err
	}
	ctx := r.Context()
	if err := s.requireOpenEvent(ctx, eventID); err != nil {
		return err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	guest, err := scanUser(tx.QueryRow(ctx,
		`INSERT INTO users AS u (email, guest, first_name, last_name, birth_date, gender, street, zip, city,
		                         emergency_name, emergency_phone, emergency_relation)
		 VALUES ($1, true, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING `+userColumns,
		email, p.FirstName, p.LastName, birth, string(p.Gender), p.Street, p.Zip, p.City,
		p.EmergencyName, p.EmergencyPhone, string(p.EmergencyRelation)))
	if isUniqueViolation(err) {
		return errorf(http.StatusConflict, "Es gibt bereits einen Benutzer mit dieser Email, bitte über die Suche einchecken")
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_members (project_id, user_id, status) VALUES ($1, $2, $3)`,
		m.Project.ID, guest.ID, string(p.Status)); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionGuestCreated,
		Target: guest}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	res, err := s.checkInUser(ctx, m, eventID, guest.ID)
	if err != nil {
		return err
	}
	res.Person = res.Person.visibleTo(m.ID, m.EmergencyAccess)
	writeJSON(w, http.StatusCreated, res)
	return nil
}

func (s *Server) requireOpenEvent(ctx context.Context, eventID int) error {
	var cancelled bool
	if err := s.db.QueryRow(ctx, `SELECT cancelled FROM events WHERE id = $1`, eventID).Scan(&cancelled); err != nil {
		return err
	}
	if cancelled {
		return errorf(http.StatusConflict, "Der Event ist abgesagt")
	}
	return nil
}

// checkInUser checkt ein Mitglied des Projekts ein. Ist es bereits eingecheckt, wird der bestehende Check-in
// geliefert. Verwarnte Personen werden eingecheckt, gesperrte nicht. Beides wird mitgeliefert und angezeigt.
func (s *Server) checkInUser(ctx context.Context, m *Member, eventID int, userID string) (*checkinResult, error) {
	res := &checkinResult{Sanctions: []Sanction{}}
	var err error
	res.Person, err = s.loadUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorf(http.StatusNotFound, "Teilnehmer nicht gefunden")
	}
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow(ctx, `SELECT status FROM project_members WHERE project_id = $1 AND user_id = $2`,
		m.Project.ID, userID).Scan(&res.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorf(http.StatusConflict,
			res.Person.Name()+" ist nicht für dieses Projekt angemeldet (über die Startseite oder den Projekt-Link anmelden)")
	}
	if err != nil {
		return nil, err
	}
	if res.Sanctions, err = s.loadSanctionsFor(ctx, m.Project.ID, userID); err != nil {
		return nil, err
	}
	if res.Sanctions == nil {
		res.Sanctions = []Sanction{}
	}

	// Anmeldung gleich mitliefern, spart beim Scannen eine Abfrage
	const registered = `EXISTS (SELECT 1 FROM registrations r WHERE r.event_id = $1 AND r.user_id = $2)`
	if banned(res.Sanctions) {
		res.Blocked = true
		err = s.db.QueryRow(ctx, `SELECT `+registered, eventID, userID).Scan(&res.Registered)
		return res, err
	}
	var at time.Time
	err = s.db.QueryRow(ctx,
		`INSERT INTO checkins (event_id, user_id, checked_in_by) VALUES ($1, $2, $3)
		 ON CONFLICT (event_id, user_id) WHERE user_id IS NOT NULL DO NOTHING
		 RETURNING id, checked_in_at, `+registered,
		eventID, userID, m.ID).Scan(&res.CheckinID, &at, &res.Registered)
	if errors.Is(err, pgx.ErrNoRows) {
		res.AlreadyCheckedIn = true
		err = s.db.QueryRow(ctx,
			`SELECT id, checked_in_at, `+registered+` FROM checkins WHERE event_id = $1 AND user_id = $2`,
			eventID, userID).Scan(&res.CheckinID, &at, &res.Registered)
	}
	res.CheckedInAt = &at
	return res, err
}

func (s *Server) deleteCheckIn(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	checkinID, err := pathInt(r, "checkinID")
	if err != nil {
		return err
	}
	if err := s.execOne(r.Context(), `DELETE FROM checkins WHERE id = $1 AND event_id = $2`, checkinID, eventID); err != nil {
		return err
	}
	return noContent(w)
}

// listParticipants liefert alle Angemeldeten und alle Eingecheckten (inkl. Gäste) eines Events,
// mit geltenden Verwarnungen und Sperren.
func (s *Server) listParticipants(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT c.id, u.id::text, trim(u.first_name || ' ' || u.last_name), COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''),
		        COALESCE(pm.status, ''), u.guest, true, c.checked_in_at
		 FROM registrations reg
		 JOIN users u ON u.id = reg.user_id
		 LEFT JOIN project_members pm ON pm.project_id = $2 AND pm.user_id = u.id
		 LEFT JOIN checkins c ON c.event_id = reg.event_id AND c.user_id = reg.user_id
		 WHERE reg.event_id = $1
		 UNION ALL
		 SELECT c.id, u.id::text,
		        COALESCE(trim(u.first_name || ' ' || u.last_name), 'Gelöschtes Konto'),
		        COALESCE(to_char(u.birth_date, 'YYYY-MM-DD'), ''), COALESCE(pm.status, ''),
		        COALESCE(u.guest, false), false, c.checked_in_at
		 FROM checkins c
		 LEFT JOIN users u ON u.id = c.user_id
		 LEFT JOIN project_members pm ON pm.project_id = $2 AND pm.user_id = u.id
		 WHERE c.event_id = $1
		   AND NOT EXISTS (SELECT 1 FROM registrations reg WHERE reg.event_id = c.event_id AND reg.user_id = c.user_id)
		 ORDER BY 3`, eventID, m.Project.ID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (participant, error) {
		var p participant
		err := row.Scan(&p.CheckinID, &p.UserID, &p.Name, &p.BirthDate, &p.Status, &p.Guest, &p.Registered, &p.CheckedInAt)
		return p, err
	})
	if err != nil {
		return err
	}
	var ids []string
	for _, p := range list {
		if p.UserID != nil {
			ids = append(ids, *p.UserID)
		}
	}
	sanctions, err := s.visibleSanctions(r.Context(), s.db, m.Project.ID, ids)
	if err != nil {
		return err
	}
	for i, p := range list {
		list[i].Sanctions = []Sanction{}
		if p.UserID != nil && sanctions[*p.UserID] != nil {
			list[i].Sanctions = sanctions[*p.UserID]
		}
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}
