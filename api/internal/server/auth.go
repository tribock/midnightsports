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
	sessionMaxAge  = 30 * 24 * time.Hour
	googleUserInfo = "https://openidconnect.googleapis.com/v1/userinfo"
)

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"devLogin":     s.cfg.DevLogin,
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
	http.SetCookie(w, s.cookie(stateCookie, state, 10*time.Minute))
	url := s.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("prompt", "select_account"))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(stateCookie)
	if err != nil || c.Value == "" || c.Value != r.URL.Query().Get("state") {
		http.Redirect(w, r, "/login?error=state", http.StatusFound)
		return
	}
	http.SetCookie(w, s.cookie(stateCookie, "", -1))

	info, err := s.fetchGoogleUser(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		slog.Warn("google login failed", "error", err)
		http.Redirect(w, r, "/login?error=google", http.StatusFound)
		return
	}
	if !info.EmailVerified {
		http.Redirect(w, r, "/login?error=email", http.StatusFound)
		return
	}
	id, err := s.upsertUser(r.Context(), info.Email, info.Sub, info.GivenName, info.FamilyName)
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

func (s *Server) fetchGoogleUser(ctx context.Context, code string) (*googleUser, error) {
	tok, err := s.oauth.Exchange(ctx, code)
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
	id, err := s.upsertUser(r.Context(), addr.Address, "", "", "")
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

// upsertUser sucht den Benutzer über die Google-ID oder Email und legt ihn bei Bedarf an.
// Die Emails aus ADMIN_EMAIL erhalten immer die Rolle Admin.
func (s *Server) upsertUser(ctx context.Context, email, sub, firstName, lastName string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var googleSub *string
	if sub != "" {
		googleSub = &sub
	}

	var id string
	err := s.db.QueryRow(ctx,
		`SELECT id FROM users WHERE ($1::text IS NOT NULL AND google_sub = $1) OR email = $2
		 ORDER BY google_sub = $1 DESC NULLS LAST LIMIT 1`,
		googleSub, email).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = s.db.QueryRow(ctx,
			`INSERT INTO users (email, google_sub, first_name, last_name) VALUES ($1, $2, $3, $4) RETURNING id`,
			email, googleSub, firstName, lastName).Scan(&id)
	case err == nil:
		_, err = s.db.Exec(ctx,
			`UPDATE users SET email = $2, google_sub = COALESCE(google_sub, $3) WHERE id = $1`,
			id, email, googleSub)
	}
	if err != nil {
		return "", err
	}
	if slices.Contains(s.cfg.AdminEmail, email) {
		_, err = s.db.Exec(ctx, `UPDATE users SET role = 'admin' WHERE id = $1`, id)
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
