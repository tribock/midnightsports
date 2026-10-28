package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

const (
	sessionCookie  = "ms_session"
	stateCookie    = "ms_oauth_state"
	verifierCookie = "ms_oauth_verifier"
	sessionMaxAge  = 30 * 24 * time.Hour
	googleUserInfo = "https://openidconnect.googleapis.com/v1/userinfo"
)

// Spalten mit der Benutzer-ID des Login-Anbieters
const (
	googleID = "google_sub"
	appleID  = "apple_sub"
)

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"devLogin":     s.cfg.DevLogin,
		"appleLogin":   s.apple != nil,
		"appleWallet":  s.appleWallet != nil,
		"googleWallet": s.googleWallet != nil,
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.cfg.GoogleClientID == "" {
		writeError(w, errorf(http.StatusServiceUnavailable, "Google Login ist nicht konfiguriert"))
		return
	}
	state := randomToken()
	// PKCE: ein abgefangener Code ist ohne den Verifier aus dem Cookie wertlos
	verifier := oauth2.GenerateVerifier()
	http.SetCookie(w, s.cookie(stateCookie, state, 10*time.Minute))
	http.SetCookie(w, s.cookie(verifierCookie, verifier, 10*time.Minute))
	url := s.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("prompt", "select_account"), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(stateCookie)
	if err != nil || c.Value == "" || c.Value != r.URL.Query().Get("state") {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}
	v, err := r.Cookie(verifierCookie)
	if err != nil || v.Value == "" {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}
	http.SetCookie(w, s.cookie(stateCookie, "", -1))
	http.SetCookie(w, s.cookie(verifierCookie, "", -1))

	info, err := s.fetchGoogleUser(r.Context(), r.URL.Query().Get("code"), v.Value)
	if err != nil {
		slog.Warn("google login failed", "error", err)
		http.Redirect(w, r, "/login?error=google", http.StatusFound)
		return
	}
	if !info.EmailVerified {
		http.Redirect(w, r, "/login?error=email", http.StatusFound)
		return
	}
	id, err := s.upsertUser(r.Context(), googleID, info.Sub, info.Email, info.GivenName, info.FamilyName)
	if err == nil {
		err = s.startSession(w, r.Context(), id)
	}
	if err != nil {
		slog.Error("login failed", "error", err)
		http.Redirect(w, r, "/login?error=server", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

type googleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

func (s *Server) fetchGoogleUser(ctx context.Context, code, verifier string) (*googleUser, error) {
	tok, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, err
	}
	resp, err := s.oauth.Client(ctx, tok).Get(googleUserInfo)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("userinfo: " + resp.Status)
	}
	var u googleUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	if u.Sub == "" || u.Email == "" {
		return nil, errors.New("userinfo: sub oder email fehlt")
	}
	return &u, nil
}

// devLogin meldet einen Benutzer nur anhand der Email an. Nur lokal mit DEV_LOGIN=true.
func (s *Server) devLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DevLogin {
		writeError(w, errNotFound)
		return
	}
	var req struct{ Email string }
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	addr, err := mail.ParseAddress(req.Email)
	if err != nil {
		writeError(w, errorf(http.StatusBadRequest, "Ungültige Email-Adresse"))
		return
	}
	id, err := s.upsertUser(r.Context(), googleID, "", addr.Address, "", "")
	if err == nil {
		err = s.startSession(w, r.Context(), id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.db.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash = $1`, s.hashToken(c.Value))
	}
	http.SetCookie(w, s.cookie(sessionCookie, "", -1))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// upsertUser sucht den Benutzer über die ID des Login-Anbieters (Spalte idColumn) oder die Email
// und legt ihn bei Bedarf an. So kann sich dieselbe Person mit Google und Apple anmelden, wenn die Email gleich ist.
// Die Emails aus ADMIN_EMAIL sind immer globale Admins.
func (s *Server) upsertUser(ctx context.Context, idColumn, sub, email, firstName, lastName string) (string, error) {
	if idColumn != googleID && idColumn != appleID {
		return "", errors.New("unbekannter Login-Anbieter: " + idColumn)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	var providerID *string
	if sub != "" {
		providerID = &sub
	}

	var id string
	err := s.db.QueryRow(ctx,
		`SELECT id FROM users WHERE ($1::text IS NOT NULL AND `+idColumn+` = $1) OR email = $2
		 ORDER BY `+idColumn+` = $1 DESC NULLS LAST LIMIT 1`,
		providerID, email).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = s.db.QueryRow(ctx,
			`INSERT INTO users (email, `+idColumn+`, first_name, last_name) VALUES ($1, $2, $3, $4) RETURNING id`,
			email, providerID, firstName, lastName).Scan(&id)
	case err == nil:
		// Vom globalen Admin angelegte Personen (Projekt-Admin) haben noch keinen Namen
		_, err = s.db.Exec(ctx,
			`UPDATE users SET email = $2, `+idColumn+` = COALESCE(`+idColumn+`, $3), guest = false,
			 first_name = CASE WHEN first_name = '' THEN $4 ELSE first_name END,
			 last_name = CASE WHEN last_name = '' THEN $5 ELSE last_name END
			 WHERE id = $1`,
			id, email, providerID, firstName, lastName)
	}
	if err != nil {
		return "", err
	}
	if slices.Contains(s.cfg.AdminEmail, email) {
		_, err = s.db.Exec(ctx, `UPDATE users SET global_admin = true WHERE id = $1`, id)
	}
	return id, err
}

func (s *Server) startSession(w http.ResponseWriter, ctx context.Context, userID string) error {
	token := randomToken()
	_, err := s.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.hashToken(token), userID, time.Now().Add(sessionMaxAge))
	if err != nil {
		return err
	}
	// Abgelaufene Sessions aufräumen
	s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	http.SetCookie(w, s.cookie(sessionCookie, token, sessionMaxAge))
	return nil
}

func (s *Server) sessionUser(r *http.Request) (*User, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil, errUnauthorized
	}
	u, err := scanUser(s.db.QueryRow(r.Context(),
		`SELECT `+userColumns+` FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > now()`,
		s.hashToken(c.Value)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errUnauthorized
	}
	return u, err
}

// hashToken speichert nur einen HMAC des Tokens. Ein neues SESSION_SECRET meldet alle Benutzer ab.
func (s *Server) hashToken(token string) []byte {
	h := hmac.New(sha256.New, s.cfg.SessionSecret)
	h.Write([]byte(token))
	return h.Sum(nil)
}

func (s *Server) cookie(name, value string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
	}
}

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
