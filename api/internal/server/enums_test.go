package server

import "testing"

func TestEnums(t *testing.T) {
	for _, g := range []Gender{"w", "m", "d"} {
		if !g.Valid() {
			t.Errorf("Gender %q muss gültig sein", g)
		}
	}
	for _, r := range []EmergencyRelation{"vater", "mutter", "andere"} {
		if !r.Valid() {
			t.Errorf("EmergencyRelation %q muss gültig sein", r)
		}
	}
	for _, s := range []UserStatus{"schueler", "gast_extern", "mitarbeiter"} {
		if !s.Valid() {
			t.Errorf("UserStatus %q muss gültig sein", s)
		}
	}

	if Gender("").Valid() || Gender("W").Valid() || Gender("x").Valid() {
		t.Error("ungültiges Geschlecht akzeptiert")
	}
	if EmergencyRelation("").Valid() || EmergencyRelation("Mutter").Valid() || EmergencyRelation("onkel").Valid() {
		t.Error("ungültige Beziehung akzeptiert")
	}
	if UserStatus("").Valid() || UserStatus("gast").Valid() || UserStatus("schueler_aadorf").Valid() {
		t.Error("ungültiger Status akzeptiert")
	}
}

func TestCheckStatusChange(t *testing.T) {
	admin, pl := RoleAdmin, RoleProjektleiter

	tests := []struct {
		name          string
		editor        string
		current, next UserStatus
		ok            bool
	}{
		{"Gast als Schüler erfassen", pl, "", StatusSchueler, true},
		{"Wechsel Schüler zu Gast", pl, StatusSchueler, StatusGastExtern, true},
		{"Mitarbeiter selbst wählen", pl, StatusGastExtern, StatusMitarbeiter, false},
		{"Gast als Mitarbeiter erfassen", pl, "", StatusMitarbeiter, false},
		{"Mitarbeiter bleibt beim Speichern", pl, StatusMitarbeiter, StatusMitarbeiter, true},
		{"Mitarbeiter selbst entfernen", pl, StatusMitarbeiter, StatusGastExtern, false},
		{"Projekt-Admin vergibt Mitarbeiter", admin, StatusGastExtern, StatusMitarbeiter, true},
		{"Projekt-Admin entfernt Mitarbeiter", admin, StatusMitarbeiter, StatusSchueler, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkStatusChange(tt.editor, tt.current, tt.next); (err == nil) != tt.ok {
				t.Errorf("err = %v, want ok = %v", err, tt.ok)
			}
		})
	}
}
