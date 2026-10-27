package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// PayoutEntry ist eine Zeile der Auszahlung an einem Event.
// Salary nil: Abrechnung über die Zeiterfassung, keine Auszahlung am Event.
type PayoutEntry struct {
	UserID       string   `json:"userId"`
	Name         string   `json:"name"`
	FunctionName string   `json:"functionName"`
	Salary       *float64 `json:"salary"`
	PaidByTime   bool     `json:"paidByTime"`
	// Signature: Unterschrift als PNG (im JSON base64-kodiert)
	Signature    []byte     `json:"signature,omitempty"`
	SignedAt     *time.Time `json:"signedAt"`
	SignedByName *string    `json:"signedByName"`
}

// Payout ist die Auszahlung an einem Event mit dem Abschluss durch die Projektleitung.
type Payout struct {
	Entries []PayoutEntry `json:"entries"`
	// Abschluss: Unterschrift der Projektleitung als PNG (im JSON base64-kodiert)
	Signature    []byte     `json:"signature,omitempty"`
	ClosedAt     *time.Time `json:"closedAt"`
	ClosedByName *string    `json:"closedByName"`
}

var errPayoutClosed = errorf(http.StatusConflict, "Die Auszahlung ist abgeschlossen")

const maxSignatureSize = 512 << 10

var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// parseSignature dekodiert eine Unterschrift (PNG als base64, optional als data-URL).
func parseSignature(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "data:image/png;base64,")
	img, err := base64.StdEncoding.DecodeString(s)
	if err != nil || !bytes.HasPrefix(img, pngMagic) {
		return nil, errorf(http.StatusBadRequest, "Ungültige Unterschrift")
	}
	if len(img) > maxSignatureSize {
		return nil, errorf(http.StatusBadRequest, "Unterschrift ist zu gross")
	}
	return img, nil
}

// listPayout liefert alle eingeteilten Personen mit Betrag und Unterschrift sowie den Abschluss.
func (s *Server) listPayout(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	ctx := r.Context()
	var p Payout
	if err := s.db.QueryRow(ctx,
		`SELECT e.payout_signature, e.payout_closed_at, NULLIF(trim(cb.first_name || ' ' || cb.last_name), '')
		 FROM events e
		 LEFT JOIN users cb ON cb.id = e.payout_closed_by
		 WHERE e.id = $1`, eventID).Scan(&p.Signature, &p.ClosedAt, &p.ClosedByName); err != nil {
		return err
	}
	rows, err := s.db.Query(ctx,
		`SELECT es.user_id::text, trim(u.first_name || ' ' || u.last_name), f.name, es.salary::float8,
		        es.salary IS NULL, es.signature, es.signed_at, NULLIF(trim(sb.first_name || ' ' || sb.last_name), '')
		 FROM event_staff es
		 JOIN users u ON u.id = es.user_id
		 JOIN functions f ON f.id = es.function_id
		 LEFT JOIN users sb ON sb.id = es.signed_by
		 WHERE es.event_id = $1
		 ORDER BY es.salary IS NULL, f.name, u.last_name, u.first_name`, eventID)
	if err != nil {
		return err
	}
	p.Entries, err = pgx.CollectRows(rows, pgx.RowToStructByPos[PayoutEntry])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

// checkPayoutOpen verhindert Änderungen an Unterschriften, nachdem die Projektleitung abgeschlossen hat.
func (s *Server) checkPayoutOpen(ctx context.Context, eventID int) error {
	var closed bool
	if err := s.db.QueryRow(ctx,
		`SELECT payout_closed_at IS NOT NULL FROM events WHERE id = $1`, eventID).Scan(&closed); err != nil {
		return err
	}
	if closed {
		return errPayoutClosed
	}
	return nil
}

// signPayout speichert die Unterschrift, mit der eine Person den Empfang ihres Betrags bestätigt.
func (s *Server) signPayout(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Signature string `json:"signature"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	img, err := parseSignature(req.Signature)
	if err != nil {
		return err
	}

	userID, err := pathUUID(r, "userId")
	if err != nil {
		return err
	}
	ctx := r.Context()
	if err := s.checkPayoutOpen(ctx, eventID); err != nil {
		return err
	}
	var paidByTime bool
	if err := s.db.QueryRow(ctx,
		`SELECT salary IS NULL FROM event_staff WHERE event_id = $1 AND user_id = $2`,
		eventID, userID).Scan(&paidByTime); err != nil {
		return err
	}
	if paidByTime {
		return errorf(http.StatusConflict, "Abrechnung über die Zeiterfassung, keine Auszahlung am Event")
	}
	err = s.execOne(ctx,
		`UPDATE event_staff SET signature = $3, signed_at = now(), signed_by = $4
		 WHERE event_id = $1 AND user_id = $2 AND signature IS NULL`,
		eventID, userID, img, u.ID)
	if errors.Is(err, errNotFound) {
		return errorf(http.StatusConflict, "Empfang wurde bereits bestätigt")
	}
	if err != nil {
		return err
	}
	return noContent(w)
}

// unsignPayout entfernt eine Unterschrift, z.B. wenn sie versehentlich erfasst wurde.
func (s *Server) unsignPayout(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	userID, err := pathUUID(r, "userId")
	if err != nil {
		return err
	}
	if err := s.checkPayoutOpen(r.Context(), eventID); err != nil {
		return err
	}
	if err := s.execOne(r.Context(),
		`UPDATE event_staff SET signature = NULL, signed_at = NULL, signed_by = NULL
		 WHERE event_id = $1 AND user_id = $2`,
		eventID, userID); err != nil {
		return err
	}
	return noContent(w)
}

// closePayout schliesst die Auszahlung mit der Unterschrift der Projektleitung ab.
// Fehlende Unterschriften im Team verhindern den Abschluss nicht, die Übersicht warnt davor.
func (s *Server) closePayout(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Signature string `json:"signature"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	img, err := parseSignature(req.Signature)
	if err != nil {
		return err
	}
	ctx := r.Context()
	if err := s.checkPayoutOpen(ctx, eventID); err != nil {
		return err
	}
	err = s.execOne(ctx,
		`UPDATE events SET payout_signature = $2, payout_closed_at = now(), payout_closed_by = $3
		 WHERE id = $1 AND payout_closed_at IS NULL`,
		eventID, img, u.ID)
	if errors.Is(err, errNotFound) {
		return errPayoutClosed
	}
	if err != nil {
		return err
	}
	return noContent(w)
}

// reopenPayout hebt den Abschluss auf, z.B. wenn eine Unterschrift nachgeholt werden muss.
func (s *Server) reopenPayout(w http.ResponseWriter, r *http.Request, u *User) error {
	eventID, err := pathInt(r, "id")
	if err != nil {
		return err
	}
	if err := s.execOne(r.Context(),
		`UPDATE events SET payout_signature = NULL, payout_closed_at = NULL, payout_closed_by = NULL
		 WHERE id = $1`, eventID); err != nil {
		return err
	}
	return noContent(w)
}
