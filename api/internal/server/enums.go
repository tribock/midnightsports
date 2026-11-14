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

// staffFunction liefert die Funktion zum Status: Mitarbeiter brauchen eine Funktion, alle anderen haben keine
// (wer den Status Mitarbeiter verliert, verliert auch die Funktion).
func staffFunction(status UserStatus, functionID *int) (*int, error) {
	if status != StatusMitarbeiter {
		return nil, nil
	}
	if functionID == nil {
		return nil, errorf(http.StatusBadRequest, "Mitarbeiter brauchen eine Funktion")
	}
	return functionID, nil
}
