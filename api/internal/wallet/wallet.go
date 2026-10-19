// Package wallet erstellt Pässe für Apple Wallet und Google Wallet mit dem QR-Code des Teilnehmers.
package wallet

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

var (
	colorBackground = color.RGBA{0x1b, 0x1f, 0x3b, 0xff}
	colorAccent     = color.RGBA{0xf5, 0xc8, 0x4c, 0xff}
)

// IconPNG zeichnet das App-Icon (Mond auf dunklem Grund) in der gewünschten Grösse.
func IconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	// Mondsichel: grosser Kreis minus versetzter Kreis
	cx, cy, r := 0.5*s, 0.5*s, 0.34*s
	ox, oy := cx+0.17*s, cy-0.12*s
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			inMoon := sq(fx-cx)+sq(fy-cy) <= r*r
			inShadow := sq(fx-ox)+sq(fy-oy) <= r*r
			if inMoon && !inShadow {
				img.Set(x, y, colorAccent)
			} else {
				img.Set(x, y, colorBackground)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func sq(v float64) float64 { return v * v }
