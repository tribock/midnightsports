package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func testIDToken(claims map[string]any) string {
	payload, _ := json.Marshal(claims)
	return "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".signatur"
}

func TestParseAppleIDToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	valid := func() map[string]any {
		return map[string]any{
			"iss": appleIssuer, "aud": "ch.midnightsports.login", "exp": now.Unix() + 600,
			"sub": "001234.abc", "nonce": "n1", "email": "anna@privaterelay.appleid.com", "email_verified": "true",
		}
	}
	tests := []struct {
		name   string
		change func(map[string]any)
		ok     bool
	}{
		{"gültig, email_verified als Text", func(map[string]any) {}, true},
		{"gültig, email_verified als Boolean", func(c map[string]any) { c["email_verified"] = true }, true},
		{"falscher Aussteller", func(c map[string]any) { c["iss"] = "https://evil.example" }, false},
		{"falscher Empfänger", func(c map[string]any) { c["aud"] = "andere.app" }, false},
		{"abgelaufen", func(c map[string]any) { c["exp"] = now.Unix() - 1 }, false},
		{"falsche Nonce", func(c map[string]any) { c["nonce"] = "n2" }, false},
		{"ohne Nonce", func(c map[string]any) { delete(c, "nonce") }, false},
		{"ohne Email", func(c map[string]any) { delete(c, "email") }, false},
		{"ohne sub", func(c map[string]any) { c["sub"] = "" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid()
			tt.change(c)
			got, err := parseAppleIDToken(testIDToken(c), "ch.midnightsports.login", "n1", now)
			if (err == nil) != tt.ok {
				t.Fatalf("err = %v, gültig erwartet: %v", err, tt.ok)
			}
			if tt.ok && !bool(got.EmailVerified) {
				t.Error("email_verified nicht erkannt")
			}
		})
	}
	if _, err := parseAppleIDToken("kein-jwt", "x", "n1", now); err == nil {
		t.Error("ungültiges Format akzeptiert")
	}
}

func TestAppleEmailNotVerified(t *testing.T) {
	for _, v := range []string{`false`, `"false"`, `null`} {
		var b appleBool = true
		if err := b.UnmarshalJSON([]byte(v)); err != nil || b {
			t.Errorf("%s: %v, %v", v, b, err)
		}
	}
}

func TestParseAppleName(t *testing.T) {
	tests := map[string][2]string{
		`{"name":{"firstName":"Anna","lastName":"Muster"},"email":"a@b.ch"}`: {"Anna", "Muster"},
		`{"name":{"firstName":" Anna "}}`:                                    {"Anna", ""},
		``:                                                                   {"", ""},
		`kein json`:                                                          {"", ""},
	}
	for in, want := range tests {
		if first, last := parseAppleName(in); first != want[0] || last != want[1] {
			t.Errorf("parseAppleName(%q) = %q %q, want %q", in, first, last, want)
		}
	}
}

func TestAppleClientSecret(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	parsed, err := parseAppleKey(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	jwt, err := appleClientSecret(parsed, "KEY123", "TEAM123", "ch.midnightsports.login", now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT mit %d Teilen", len(parts))
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	if len(sig) != 64 || !ecdsa.Verify(&key.PublicKey, hash[:], r, s) {
		t.Error("Signatur ungültig")
	}
	var header, claims map[string]any
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	c, _ := base64.RawURLEncoding.DecodeString(parts[1])
	json.Unmarshal(h, &header)
	json.Unmarshal(c, &claims)
	if header["alg"] != "ES256" || header["kid"] != "KEY123" {
		t.Errorf("Header %v", header)
	}
	if claims["iss"] != "TEAM123" || claims["sub"] != "ch.midnightsports.login" || claims["aud"] != appleIssuer {
		t.Errorf("Claims %v", claims)
	}
}

func TestParseAppleKeyInvalid(t *testing.T) {
	if _, err := parseAppleKey([]byte("kein pem")); err == nil {
		t.Error("ungültiger Schlüssel akzeptiert")
	}
}
