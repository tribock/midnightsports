package server

import (
	"bytes"
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

// listPayout liefert alle eingeteilten Personen mit Betrag und Unterschrift.
func (s *Server) listPayout(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(),
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
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[PayoutEntry])
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

// signPayout speichert die Unterschrift, mit der eine Person den Empfang ihres Betrags bestätigt.
func (s *Server) signPayout(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
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
		eventID, userID, img, m.ID)
	if errors.Is(err, errNotFound) {
		return errorf(http.StatusConflict, "Empfang wurde bereits bestätigt")
	}
	if err != nil {
		return err
	}
	return noContent(w)
}

// unsignPayout entfernt eine Unterschrift, z.B. wenn sie versehentlich erfasst wurde.
func (s *Server) unsignPayout(w http.ResponseWriter, r *http.Request, m *Member) error {
	eventID, err := s.projectEvent(r, m)
	if err != nil {
		return err
	}
	userID, err := pathUUID(r, "userId")
	if err != nil {
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
