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

// UserStatus beschreibt, woher ein Teilnehmer kommt. Mitarbeiter vergeben nur Admins.
type UserStatus string

const (
	StatusSchuelerAadorf UserStatus = "schueler_aadorf"
	StatusGastExtern     UserStatus = "gast_extern"
	StatusMitarbeiter    UserStatus = "mitarbeiter"
)

func (s UserStatus) Valid() bool {
	return slices.Contains([]UserStatus{StatusSchuelerAadorf, StatusGastExtern, StatusMitarbeiter}, s)
}

// checkStatusChange: den Status Mitarbeiter vergeben und entziehen nur Admins.
// Alle anderen dürfen ihn unverändert lassen, z.B. beim Speichern des eigenen Profils.
func checkStatusChange(editor *User, current, next UserStatus) error {
	if editor.Role == RoleAdmin || current == next {
		return nil
	}
	if current == StatusMitarbeiter || next == StatusMitarbeiter {
		return errorf(http.StatusForbidden, "Den Status Mitarbeiter vergeben nur Admins")
	}
	return nil
}
