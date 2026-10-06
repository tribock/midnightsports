// Package config lädt die Konfiguration aus Umgebungsvariablen.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	OAuthRedirectURL   string
	SessionSecret      []byte
	QRSigningKey       []byte
	AdminEmail         []string
	DevLogin           bool
	StaticDir          string

	// Apple Wallet (optional)
	ApplePassTypeID  string
	AppleTeamID      string
	ApplePassPEMFile string

	// Google Wallet (optional)
	GoogleWalletIssuerID string
	GoogleWalletKeyFile  string
}

// Load liest die Konfiguration. Eine vorhandene .env-Datei wird zuerst geladen,
// bereits gesetzte Umgebungsvariablen haben Vorrang.
func Load(envFiles ...string) (*Config, error) {
	for _, f := range envFiles {
		if err := loadEnvFile(f); err != nil {
			return nil, err
		}
	}

	c := &Config{
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		OAuthRedirectURL:   os.Getenv("OAUTH_REDIRECT_URL"),
		SessionSecret:      []byte(os.Getenv("SESSION_SECRET")),
		QRSigningKey:       []byte(os.Getenv("QR_SIGNING_KEY")),
		AdminEmail:         splitEmails(os.Getenv("ADMIN_EMAIL")),
		DevLogin:           os.Getenv("DEV_LOGIN") == "true",
		StaticDir:          os.Getenv("STATIC_DIR"),

		ApplePassTypeID:  os.Getenv("APPLE_PASS_TYPE_ID"),
		AppleTeamID:      os.Getenv("APPLE_TEAM_ID"),
		ApplePassPEMFile: os.Getenv("APPLE_PASS_PEM_FILE"),

		GoogleWalletIssuerID: os.Getenv("GOOGLE_WALLET_ISSUER_ID"),
		GoogleWalletKeyFile:  os.Getenv("GOOGLE_WALLET_KEY_FILE"),
	}
	return c, c.validate()
}

func (c *Config) validate() error {
	var errs []error
	require := func(name, value string) {
		if value == "" {
			errs = append(errs, fmt.Errorf("%s fehlt", name))
		}
	}
	require("DATABASE_URL", c.DatabaseURL)
	require("OAUTH_REDIRECT_URL", c.OAuthRedirectURL)
	if u, err := url.Parse(c.OAuthRedirectURL); c.OAuthRedirectURL != "" && (err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https")) {
		errs = append(errs, errors.New("OAUTH_REDIRECT_URL muss eine vollständige URL sein, z.B. https://example.ch/api/auth/callback"))
	}
	if len(c.SessionSecret) < 32 {
		errs = append(errs, errors.New("SESSION_SECRET muss mindestens 32 Zeichen lang sein"))
	}
	if len(c.QRSigningKey) < 32 {
		errs = append(errs, errors.New("QR_SIGNING_KEY muss mindestens 32 Zeichen lang sein"))
	}
	if !c.DevLogin {
		require("GOOGLE_CLIENT_ID", c.GoogleClientID)
		require("GOOGLE_CLIENT_SECRET", c.GoogleClientSecret)
	}
	if c.DevLogin && !c.IsLocal() {
		errs = append(errs, errors.New("DEV_LOGIN ist nur mit einer localhost OAUTH_REDIRECT_URL erlaubt"))
	}
	return errors.Join(errs...)
}

// IsLocal ist true, wenn die App unter localhost läuft.
func (c *Config) IsLocal() bool {
	u, err := url.Parse(c.OAuthRedirectURL)
	if err != nil {
		return false
	}
	h := u.Hostname()
	return h == "localhost" || h == "127.0.0.1"
}

// BaseURL ist die öffentliche Adresse der App, z.B. https://midnightsports.example
func (c *Config) BaseURL() string {
	u, err := url.Parse(c.OAuthRedirectURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// SecureCookies ist true, wenn die App über HTTPS erreichbar ist.
func (c *Config) SecureCookies() bool {
	return strings.HasPrefix(c.OAuthRedirectURL, "https://")
}

// splitEmails liest eine komma-getrennte Liste von Email-Adressen. Leere Einträge werden ignoriert.
func splitEmails(s string) []string {
	var emails []string
	for _, e := range strings.Split(s, ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			emails = append(emails, e)
		}
	}
	return emails
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}
	return s.Err()
}
