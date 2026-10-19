package config

import (
	"slices"
	"strings"
	"testing"
)

func TestValidateRedirectURL(t *testing.T) {
	base := Config{
		DatabaseURL:        "postgres://localhost/db",
		GoogleClientID:     "id",
		GoogleClientSecret: "secret",
		SessionSecret:      []byte(strings.Repeat("s", 32)),
		QRSigningKey:       []byte(strings.Repeat("q", 32)),
	}
	tests := map[string]bool{
		"https://midnightsports.example/api/auth/callback": true,
		"http://localhost:5173/api/auth/callback":          true,
		"/api/auth/callback":                               false,
		"midnightsports.example/api/auth/callback":         false,
		"ftp://midnightsports.example/api/auth/callback":   false,
	}
	for url, valid := range tests {
		c := base
		c.OAuthRedirectURL = url
		if err := c.validate(); (err == nil) != valid {
			t.Errorf("%s: err = %v, gültig erwartet: %v", url, err, valid)
		}
	}
}

func TestSplitEmails(t *testing.T) {
	tests := map[string][]string{
		"":                           nil,
		"Admin@Test.ch":              {"admin@test.ch"},
		"a@test.ch, B@test.ch":       {"a@test.ch", "b@test.ch"},
		" a@test.ch ,, b@test.ch , ": {"a@test.ch", "b@test.ch"},
	}
	for in, want := range tests {
		if got := splitEmails(in); !slices.Equal(got, want) {
			t.Errorf("splitEmails(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateAppleLogin(t *testing.T) {
	base := Config{
		DatabaseURL:        "postgres://localhost/db",
		GoogleClientID:     "id",
		GoogleClientSecret: "secret",
		OAuthRedirectURL:   "https://midnightsports.example/api/auth/callback",
		SessionSecret:      []byte(strings.Repeat("s", 32)),
		QRSigningKey:       []byte(strings.Repeat("q", 32)),
	}
	complete := func(c *Config) {
		c.AppleLoginClientID, c.AppleLoginKeyID, c.AppleLoginKeyFile, c.AppleTeamID = "ch.ms.login", "KEY", "key.p8", "TEAM"
	}
	tests := []struct {
		name   string
		change func(*Config)
		ok     bool
	}{
		{"ohne Apple", func(*Config) {}, true},
		{"vollständig", complete, true},
		{"ohne Team ID", func(c *Config) { complete(c); c.AppleTeamID = "" }, false},
		{"nur Schlüssel", func(c *Config) { c.AppleLoginKeyFile = "key.p8" }, false},
		{"ohne HTTPS", func(c *Config) { complete(c); c.OAuthRedirectURL = "http://localhost:5173/api/auth/callback" }, false},
	}
	for _, tt := range tests {
		c := base
		tt.change(&c)
		if err := c.validate(); (err == nil) != tt.ok {
			t.Errorf("%s: err = %v, gültig erwartet: %v", tt.name, err, tt.ok)
		}
	}
}
