package server

import "testing"

func TestVisibleTo(t *testing.T) {
	person := &User{ID: "p", EmergencyName: "Anna", EmergencyPhone: "079", EmergencyRelation: RelationMutter}

	tests := []struct {
		name    string
		viewer  *User
		visible bool
	}{
		{"mit Zugriff", &User{ID: "sc", EmergencyAccess: true}, true},
		{"ohne Zugriff", &User{ID: "ma"}, false},
		{"eigene Daten", &User{ID: "p"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := person.visibleTo(tt.viewer)
			if (got.EmergencyPhone != "") != tt.visible {
				t.Errorf("Notfallkontakt sichtbar = %v, want %v", got.EmergencyPhone != "", tt.visible)
			}
		})
	}
	if person.EmergencyPhone == "" {
		t.Error("visibleTo darf das Original nicht verändern")
	}
}

func TestValidateProfile(t *testing.T) {
	valid := func() User {
		return User{FirstName: " Anna ", LastName: "Muster", BirthDate: "2010-11-07", Gender: GenderW,
			Status: StatusSchuelerAadorf, Zip: "8355", City: "Aadorf",
			EmergencyName: "Eva", EmergencyPhone: "079", EmergencyRelation: RelationMutter}
	}

	p := valid()
	if _, err := validateProfile(&p); err != nil {
		t.Fatalf("gültiges Profil abgelehnt: %v", err)
	}
	if p.FirstName != "Anna" {
		t.Errorf("Leerzeichen nicht entfernt: %q", p.FirstName)
	}

	tests := []struct {
		name   string
		change func(*User)
	}{
		{"Pflichtfeld leer", func(u *User) { u.City = "  " }},
		{"Geschlecht fehlt", func(u *User) { u.Gender = "" }},
		{"Status ungültig", func(u *User) { u.Status = "lehrer" }},
		{"Beziehung fehlt", func(u *User) { u.EmergencyRelation = "" }},
		{"Geburtsdatum in der Zukunft", func(u *User) { u.BirthDate = "2999-01-01" }},
		{"Geburtsdatum ungültig", func(u *User) { u.BirthDate = "07.11.2010" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid()
			tt.change(&p)
			if _, err := validateProfile(&p); err == nil {
				t.Error("ungültiges Profil akzeptiert")
			}
		})
	}
}

func TestOptionalEmail(t *testing.T) {
	if e, err := optionalEmail("  "); e != nil || err != nil {
		t.Errorf("leere Email: got %v, %v", e, err)
	}
	if e, err := optionalEmail(" Anna@Example.CH "); err != nil || *e != "anna@example.ch" {
		t.Errorf("Email nicht normalisiert: got %v, %v", e, err)
	}
	if _, err := optionalEmail("keine-email"); err == nil {
		t.Error("ungültige Email akzeptiert")
	}
}
