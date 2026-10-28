package server

import (
	"encoding/base64"
	"testing"
)

func TestValidateProjectInput(t *testing.T) {
	salary := func(v float64) *float64 { return &v }
	valid := func() projectInput {
		return projectInput{
			Slug: " Pleno-Frauenfeld ", Name: " Pleno ", Region: "Frauenfeld", AdminEmail: " Admin@Pleno.CH ",
			Functions: []Function{{Name: "JC", Salary: salary(20)}, {Name: "PL"}},
			Location:  Location{Name: "Jugendtreff"},
			ShareWith: []string{"aadorf"},
		}
	}
	in := valid()
	if err := validateProjectInput(&in); err != nil {
		t.Fatalf("gültiges Projekt abgelehnt: %v", err)
	}
	if in.Slug != "pleno-frauenfeld" || in.Name != "Pleno" || in.AdminEmail != "admin@pleno.ch" {
		t.Errorf("nicht normalisiert: %+v", in)
	}
	tests := []struct {
		name   string
		change func(*projectInput)
	}{
		{"Name fehlt", func(p *projectInput) { p.Name = "" }},
		{"Kurzname mit Leerzeichen", func(p *projectInput) { p.Slug = "pleno frauenfeld" }},
		{"Kurzname mit Umlaut", func(p *projectInput) { p.Slug = "zürich" }},
		{"Region fehlt", func(p *projectInput) { p.Region = "" }},
		{"Admin-Email ungültig", func(p *projectInput) { p.AdminEmail = "admin" }},
		{"keine Funktion", func(p *projectInput) { p.Functions = nil }},
		{"Funktion doppelt", func(p *projectInput) { p.Functions = append(p.Functions, Function{Name: "jc"}) }},
		{"negativer Lohn", func(p *projectInput) { p.Functions[0].Salary = salary(-1) }},
		{"ohne Ort", func(p *projectInput) { p.Location.Name = " " }},
		{"teilt mit sich selbst", func(p *projectInput) { p.ShareWith = []string{"pleno-frauenfeld"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid()
			tt.change(&p)
			if err := validateProjectInput(&p); err == nil {
				t.Error("ungültiges Projekt akzeptiert")
			}
		})
	}
}

func TestParseLogo(t *testing.T) {
	b64 := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	tests := []struct {
		name     string
		in       string
		wantType string
	}{
		{"PNG", b64(append(pngMagic, 1, 2, 3)), "image/png"},
		{"PNG als data-URL", "data:image/png;base64," + b64(append(pngMagic, 1)), "image/png"},
		{"JPEG", b64([]byte{0xFF, 0xD8, 0xFF, 0xE0}), "image/jpeg"},
		{"WebP", b64([]byte("RIFF\x00\x00\x00\x00WEBPVP8 ")), "image/webp"},
		{"SVG nicht erlaubt", b64([]byte("<svg onload=alert(1)>")), ""},
		{"kein base64", "!!!", ""},
		{"zu gross", b64(append(pngMagic, make([]byte, maxLogoSize)...)), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, typ, err := parseLogo(tt.in)
			if (err == nil) != (tt.wantType != "") || typ != tt.wantType {
				t.Errorf("Typ = %q, err = %v, want %q", typ, err, tt.wantType)
			}
		})
	}
}
