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

func TestStaffFunction(t *testing.T) {
	fn := 3

	tests := []struct {
		name       string
		status     UserStatus
		functionID *int
		want       *int
		ok         bool
	}{
		{"Mitarbeiter mit Funktion", StatusMitarbeiter, &fn, &fn, true},
		{"Mitarbeiter ohne Funktion", StatusMitarbeiter, nil, nil, false},
		{"Schüler ohne Funktion", StatusSchueler, nil, nil, true},
		{"Gast extern verliert die Funktion", StatusGastExtern, &fn, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := staffFunction(tt.status, tt.functionID)
			if (err == nil) != tt.ok {
				t.Fatalf("err = %v, want ok = %v", err, tt.ok)
			}
			if got != tt.want {
				t.Errorf("Funktion = %v, want %v", got, tt.want)
			}
		})
	}
}
