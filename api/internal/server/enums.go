package server

import (
	"net/http"
	"slices"
)

// Gender ist das Geschlecht eines Benutzers.
type Gender string

const (
	GenderW Gender = "w"
	GenderM Gender = "m"
	GenderD Gender = "d"
)

func (g Gender) Valid() bool { return slices.Contains([]Gender{GenderW, GenderM, GenderD}, g) }

// EmergencyRelation ist die Beziehung des Notfallkontakts zum Benutzer.
type EmergencyRelation string

const (
	RelationVater  EmergencyRelation = "vater"
	RelationMutter EmergencyRelation = "mutter"
	RelationAndere EmergencyRelation = "andere"
)

func (r EmergencyRelation) Valid() bool {
	return slices.Contains([]EmergencyRelation{RelationVater, RelationMutter, RelationAndere}, r)
}

// UserStatus beschreibt, woher ein Mitglied im Projekt kommt (pro Projekt, gewählt beim Anmelden).
// "schueler": Schüler/in aus der Region des Projekts. Mitarbeiter vergeben nur Projekt-Admins.
type UserStatus string

const (
	StatusSchueler    UserStatus = "schueler"
	StatusGastExtern  UserStatus = "gast_extern"
	StatusMitarbeiter UserStatus = "mitarbeiter"
)

func (s UserStatus) Valid() bool {
	return slices.Contains([]UserStatus{StatusSchueler, StatusGastExtern, StatusMitarbeiter}, s)
}

// checkStatusChange: den Status Mitarbeiter vergeben und entziehen nur Projekt-Admins.
// Alle anderen dürfen ihn unverändert lassen.
func checkStatusChange(editorRole string, current, next UserStatus) error {
	if editorRole == RoleAdmin || current == next {
		return nil
	}
	if current == StatusMitarbeiter || next == StatusMitarbeiter {
		return errorf(http.StatusForbidden, "Den Status Mitarbeiter vergeben nur Projekt-Admins")
	}
	return nil
}
