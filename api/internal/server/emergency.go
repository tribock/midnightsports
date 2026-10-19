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

func (s *Server) getEmergencyAccess(w http.ResponseWriter, r *http.Request, u *User) error {
	var a EmergencyAccess
	err := s.db.QueryRow(r.Context(),
		`SELECT COALESCE((SELECT array_agg(role ORDER BY role) FROM emergency_access_roles), '{}'),
		        COALESCE((SELECT array_agg(function_id ORDER BY function_id) FROM emergency_access_functions), '{}')`,
	).Scan(&a.Roles, &a.FunctionIDs)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, a)
	return nil
}

// setEmergencyAccess ersetzt die Rollen und Funktionen mit Zugriff auf Notfallkontakte.
func (s *Server) setEmergencyAccess(w http.ResponseWriter, r *http.Request, u *User) error {
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
	batch.Queue(`DELETE FROM emergency_access_roles`)
	batch.Queue(`DELETE FROM emergency_access_functions`)
	batch.Queue(`INSERT INTO emergency_access_roles (role) SELECT DISTINCT unnest($1::text[])`, a.Roles)
	batch.Queue(`INSERT INTO emergency_access_functions (function_id) SELECT DISTINCT unnest($1::int[])`, a.FunctionIDs)
	err = tx.SendBatch(ctx, batch).Close()
	if isForeignKeyViolation(err) {
		return errorf(http.StatusBadRequest, "Unbekannte Funktion")
	}
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return s.getEmergencyAccess(w, r, u)
}
