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
