package server

import "strings"

// csvText schützt Freitext vor Formel-Injection: Excel führt Zellen, die mit = + - @ beginnen, als Formel aus
// (z.B. ein Name "=HYPERLINK(...)"). Ein vorangestelltes ' macht sie zu Text.
func csvText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
