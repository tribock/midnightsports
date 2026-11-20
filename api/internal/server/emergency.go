package server

import (
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"
)

// EmergencyAccess legt fest, welche Rollen und Funktionen Notfallkontakte einsehen dürfen.
// Admins haben immer Zugriff und sind hier nicht aufgeführt.
type EmergencyAccess struct {
	Roles       []string `json:"roles"`
	FunctionIDs []int    `json:"functionIds"`
}

// Rollen, deren Zugriff auf Notfallkontakte wählbar ist. Admins immer, Teilnehmer nie.
var emergencyAccessRoles = []string{RoleProjektleiter, RoleMitarbeiter}

// validateAccessRoles lässt nur Rollen zu, deren Zugriff wählbar ist (nicht Admin, nicht Teilnehmer).
func validateAccessRoles(roles []string) error {
	for _, role := range roles {
		if !slices.Contains(emergencyAccessRoles, role) {
			return errorf(http.StatusBadRequest, "Diese Rolle kann keinen Zugriff auf Notfallkontakte erhalten")
		}
	}
	return nil
}

func (s *Server) getEmergencyAccess(w http.ResponseWriter, r *http.Request, m *Member) error {
	var a EmergencyAccess
	err := s.db.QueryRow(r.Context(),
		`SELECT COALESCE((SELECT array_agg(role ORDER BY role) FROM emergency_access_roles WHERE project_id = $1), '{}'),
		        COALESCE((SELECT array_agg(af.function_id ORDER BY af.function_id)
		                  FROM emergency_access_functions af JOIN functions f ON f.id = af.function_id
		                  WHERE f.project_id = $1), '{}')`, m.Project.ID,
	).Scan(&a.Roles, &a.FunctionIDs)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, a)
	return nil
}

// setEmergencyAccess ersetzt die Rollen und Funktionen mit Zugriff auf Notfallkontakte des Projekts.
// Wer Notfallkontakte sehen darf, ist eine Berechtigung auf Personendaten und wird protokolliert.
func (s *Server) setEmergencyAccess(w http.ResponseWriter, r *http.Request, m *Member) error {
	var a EmergencyAccess
	if err := decode(r, &a); err != nil {
		return err
	}
	if err := validateAccessRoles(a.Roles); err != nil {
		return err
	}

	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	batch.Queue(`DELETE FROM emergency_access_roles WHERE project_id = $1`, m.Project.ID)
	batch.Queue(`DELETE FROM emergency_access_functions af USING functions f
		WHERE f.id = af.function_id AND f.project_id = $1`, m.Project.ID)
	batch.Queue(`INSERT INTO emergency_access_roles (project_id, role) SELECT DISTINCT $1::int, unnest($2::text[])`,
		m.Project.ID, a.Roles)
	// Nur Funktionen dieses Projekts
	batch.Queue(`INSERT INTO emergency_access_functions (function_id)
		SELECT DISTINCT f.id FROM functions f WHERE f.project_id = $1 AND f.id = ANY($2::int[])`,
		m.Project.ID, a.FunctionIDs)
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}
	if err := s.audit(ctx, tx, auditEntry{Actor: m.User, ProjectID: &m.Project.ID, Action: actionEmergencyAccess,
		Details: map[string]any{"rollen": a.Roles, "funktionen": a.FunctionIDs}}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return s.getEmergencyAccess(w, r, m)
}
