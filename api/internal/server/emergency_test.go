package server

import "testing"

func TestValidateAccessRoles(t *testing.T) {
	if err := validateAccessRoles([]string{RoleProjektleiter, RoleMitarbeiter}); err != nil {
		t.Errorf("wählbare Rollen abgelehnt: %v", err)
	}
	if err := validateAccessRoles(nil); err != nil {
		t.Errorf("leere Auswahl abgelehnt: %v", err)
	}
	// Admins haben immer Zugriff, Teilnehmer nie: beides ist nicht wählbar
	for _, role := range []string{RoleAdmin, RoleTeilnehmer, "chef"} {
		if validateAccessRoles([]string{role}) == nil {
			t.Errorf("Rolle %q darf nicht wählbar sein", role)
		}
	}
}
