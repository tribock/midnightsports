package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/tribock/midnightsports/api/internal/config"
)

const (
	appleIssuer      = "https://appleid.apple.com"
	appleStateCookie = "ms_apple_state"
	appleNonceCookie = "ms_apple_nonce"
	appleCookiePath  = "/api/auth/apple"
)

// appleLogin enthält die Konfiguration für den Login mit Apple.
type appleLogin struct {
	oauth  *oauth2.Config // ohne ClientSecret, das wird pro Anmeldung erzeugt
	teamID string
	keyID  string
	key    *ecdsa.PrivateKey
}

// newAppleLogin liest den Schlüssel (.p8) aus der Konfiguration. Ohne APPLE_LOGIN_CLIENT_ID ist der Login aus (nil).
func newAppleLogin(cfg *config.Config) (*appleLogin, error) {
	if !cfg.AppleLogin() {
		return nil, nil
	}
	data, err := os.ReadFile(cfg.AppleLoginKeyFile)
	if err != nil {
		return nil, fmt.Errorf("APPLE_LOGIN_KEY_FILE: %w", err)
	}
	key, err := parseAppleKey(data)
	if err != nil {
		return nil, fmt.Errorf("APPLE_LOGIN_KEY_FILE: %w", err)
	}
	return &appleLogin{
		oauth: &oauth2.Config{
			ClientID:    cfg.AppleLoginClientID,
			RedirectURL: cfg.AppleRedirectURL(),
			Scopes:      []string{"name", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   appleIssuer + "/auth/authorize",
				TokenURL:  appleIssuer + "/auth/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		teamID: cfg.AppleTeamID,
		keyID:  cfg.AppleLoginKeyID,
		key:    key,
	}, nil
}

func parseAppleKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("keine PEM-Datei")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("kein EC-Schlüssel")
	}
	return key, nil
}

func (s *Server) appleLoginStart(w http.ResponseWriter, r *http.Request) {
	if s.apple == nil {
		writeError(w, errorf(http.StatusServiceUnavailable, "Login mit Apple ist nicht konfiguriert"))
		return
	}
	state, nonce := randomToken(), randomToken()
	http.SetCookie(w, s.appleCookie(appleStateCookie, state, 10*time.Minute))
	http.SetCookie(w, s.appleCookie(appleNonceCookie, nonce, 10*time.Minute))
	// Apple liefert Name und Email nur per form_post. Die Nonce im id_token bindet den Code an diesen Browser
	url := s.apple.oauth.AuthCodeURL(state,
		oauth2.SetAuthURLParam("response_mode", "form_post"),
		oauth2.SetAuthURLParam("nonce", nonce))
	http.Redirect(w, r, url, http.StatusFound)
}

// appleCallback empfängt den POST von appleid.apple.com. Die Route ist vom CSRF-Schutz ausgenommen,
// stattdessen prüfen State und Nonce, dass die Anmeldung in diesem Browser begonnen wurde.
func (s *Server) appleCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(code string) { http.Redirect(w, r, "/login?error="+code, http.StatusSeeOther) }
	if s.apple == nil {
		fail("apple")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		fail("apple")
		return
	}
	if r.PostForm.Get("error") == "user_cancelled_authorize" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	state, err := r.Cookie(appleStateCookie)
	if err != nil || state.Value == "" || state.Value != r.PostForm.Get("state") {
		fail("state")
		return
	}
	nonce, err := r.Cookie(appleNonceCookie)
	if err != nil || nonce.Value == "" {
		fail("state")
		return
	}
	http.SetCookie(w, s.appleCookie(appleStateCookie, "", -1))
	http.SetCookie(w, s.appleCookie(appleNonceCookie, "", -1))

	claims, err := s.fetchAppleUser(r.Context(), r.PostForm.Get("code"), nonce.Value)
	if err != nil {
		slog.Warn("apple login failed", "error", err)
		fail("apple")
		return
	}
	if !claims.EmailVerified {
		fail("email")
		return
	}
	// Den Namen schickt Apple nur bei der ersten Anmeldung
	first, last := parseAppleName(r.PostForm.Get("user"))
	id, err := s.upsertUser(r.Context(), appleID, claims.Sub, claims.Email, first, last)
	if err == nil {
		err = s.startSession(w, r.Context(), id)
	}
	if err != nil {
		slog.Error("login failed", "error", err)
		fail("server")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// fetchAppleUser tauscht den Code gegen das id_token. Das id_token aus dem Formular wird ignoriert,
// weil es über den Browser kommt und gefälscht sein kann.
func (s *Server) fetchAppleUser(ctx context.Context, code, nonce string) (*appleClaims, error) {
	secret, err := appleClientSecret(s.apple.key, s.apple.keyID, s.apple.teamID, s.apple.oauth.ClientID, time.Now())
	if err != nil {
		return nil, err
	}
	conf := *s.apple.oauth
	conf.ClientSecret = secret
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	idToken, _ := tok.Extra("id_token").(string)
	// Das Token kommt direkt per TLS vom Token-Endpunkt von Apple, deshalb genügt das als Nachweis
	// des Absenders (OpenID Connect Core 3.1.3.7), die Signatur muss nicht geprüft werden.
	return parseAppleIDToken(idToken, s.apple.oauth.ClientID, nonce, time.Now())
}

type appleClaims struct {
	Iss           string    `json:"iss"`
	Aud           string    `json:"aud"`
	Exp           int64     `json:"exp"`
	Sub           string    `json:"sub"`
	Nonce         string    `json:"nonce"`
	Email         string    `json:"email"`
	EmailVerified appleBool `json:"email_verified"`
}

// appleBool: Apple liefert Booleans je nach Fall als true oder als "true".
type appleBool bool

func (b *appleBool) UnmarshalJSON(data []byte) error {
	*b = appleBool(string(data) == "true" || string(data) == `"true"`)
	return nil
}

// parseAppleIDToken liest die Claims des id_token und prüft Aussteller, Empfänger, Ablauf und Nonce.
func parseAppleIDToken(token, clientID, nonce string, now time.Time) (*appleClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("id_token: ungültiges Format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id_token: %w", err)
	}
	var c appleClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("id_token: %w", err)
	}
	switch {
	case c.Iss != appleIssuer:
		return nil, errors.New("id_token: falscher Aussteller")
	case c.Aud != clientID:
		return nil, errors.New("id_token: falscher Empfänger")
	case now.Unix() >= c.Exp:
		return nil, errors.New("id_token: abgelaufen")
	case c.Nonce == "" || c.Nonce != nonce:
		return nil, errors.New("id_token: falsche Nonce")
	case c.Sub == "" || c.Email == "":
		return nil, errors.New("id_token: sub oder email fehlt")
	}
	return &c, nil
}

// parseAppleName liest den Namen aus dem Formularfeld user (JSON, nur bei der ersten Anmeldung).
func parseAppleName(user string) (first, last string) {
	var u struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
	}
	if json.Unmarshal([]byte(user), &u) != nil {
		return "", ""
	}
	return strings.TrimSpace(u.Name.FirstName), strings.TrimSpace(u.Name.LastName)
}

// appleClientSecret erzeugt das Client-Secret: ein mit dem Schlüssel aus dem Apple-Konto signiertes JWT (ES256).
// Es ist nur kurz gültig, weil es bei jeder Anmeldung neu erzeugt wird.
func appleClientSecret(key *ecdsa.PrivateKey, keyID, teamID, clientID string, now time.Time) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": keyID})
	claims, _ := json.Marshal(map[string]any{
		"iss": teamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": appleIssuer,
		"sub": clientID,
	})
	enc := base64.RawURLEncoding
	signed := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)
	hash := sha256.Sum256([]byte(signed))
	der, err := key.Sign(rand.Reader, hash[:], nil)
	if err != nil {
		return "", err
	}
	// JWS erwartet r und s aneinandergehängt (je 32 Bytes), nicht ASN.1
	var sig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(der, &sig); err != nil {
		return "", err
	}
	raw := make([]byte, 64)
	sig.R.FillBytes(raw[:32])
	sig.S.FillBytes(raw[32:])
	return signed + "." + enc.EncodeToString(raw), nil
}

// appleCookie: Apple ruft den Callback per POST von einer fremden Seite auf. SameSite=Lax-Cookies
// würden dabei nicht mitgeschickt, deshalb SameSite=None (nur mit Secure erlaubt).
func (s *Server) appleCookie(name, value string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     appleCookiePath,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
}
