// Package wallet erstellt Pässe für Apple Wallet und Google Wallet mit dem QR-Code des Teilnehmers.
package wallet

import (
	"embed"
	"strings"
)

// Pass enthält die Angaben, die auf dem Pass erscheinen.
type Pass struct {
	UserID    string
	Token     string
	FirstName string
	LastName  string
	BirthDate string // YYYY-MM-DD
}

func (p Pass) name() string { return strings.TrimSpace(p.FirstName + " " + p.LastName) }

// birthDate formatiert das Geburtsdatum als TT.MM.JJJJ.
func (p Pass) birthDate() string {
	parts := strings.Split(p.BirthDate, "-")
	if len(parts) != 3 {
		return p.BirthDate
	}
	return parts[2] + "." + parts[1] + "." + parts[0]
}

// Logos und Icons, erzeugt mit `make logos` aus docs/logo
//
//go:embed assets/*.png
var assets embed.FS

// asset liefert eine eingebettete PNG-Datei aus assets.
func asset(name string) []byte {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		panic(err)
	}
	return b
}

// GoogleLogoPNG ist das Logo, das Google Wallet über die öffentliche URL abruft (rund zugeschnitten, 660 px).
func GoogleLogoPNG() []byte { return asset("icon-660.png") }
