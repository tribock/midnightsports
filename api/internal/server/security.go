package server

import "net/http"

// Content-Security-Policy: Skripte, Styles und Verbindungen nur von der eigenen Adresse.
// img-src data: für die Unterschriften der Auszahlung, worker-src blob: für den QR-Scanner.
// Inline-Styles erlaubt, weil Bibliotheken (QR-Scanner) Styles direkt setzen.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// secure versieht alle Antworten mit Sicherheits-Headern und blockiert Cross-Site-Anfragen,
// die etwas verändern (POST, PUT, DELETE von fremden Seiten, CSRF). Ergänzt das SameSite-Cookie.
func secure(next http.Handler, https bool) http.Handler {
	csrf := http.NewCrossOriginProtection()
	// Apple schickt den Login per POST von appleid.apple.com. State und Nonce schützen diese Route
	csrf.AddInsecureBypassPattern("POST /api/auth/apple/callback")
	protected := csrf.Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		// Kamera nur für den QR-Scanner der eigenen Seite
		h.Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if https {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		protected.ServeHTTP(w, r)
	})
}
