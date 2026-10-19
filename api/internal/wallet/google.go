package wallet

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const googleSaveURL = "https://pay.google.com/gp/v/save/"

// Google erstellt "Zu Google Wallet hinzufügen"-Links. Klasse und Pass stecken im signierten JWT,
// es sind keine Aufrufe der Wallet API nötig.
type Google struct {
	issuerID string
	email    string
	key      *rsa.PrivateKey
	origin   string
}

// NewGoogle lädt den Schlüssel des Service Accounts (JSON-Datei). Ohne Konfiguration wird nil zurückgegeben.
func NewGoogle(issuerID, keyFile, origin string) (*Google, error) {
	if issuerID == "" && keyFile == "" {
		return nil, nil
	}
	if issuerID == "" || keyFile == "" {
		return nil, errors.New("Google Wallet: GOOGLE_WALLET_ISSUER_ID und GOOGLE_WALLET_KEY_FILE sind beide nötig")
	}
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("Google Wallet: %w", err)
	}
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
	}
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, fmt.Errorf("Google Wallet: Service-Account-Datei: %w", err)
	}
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil || sa.ClientEmail == "" {
		return nil, errors.New("Google Wallet: Service-Account-Datei ist unvollständig")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Google Wallet: %w", err)
	}
	key, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Google Wallet: RSA-Schlüssel erwartet")
	}
	return &Google{issuerID: issuerID, email: sa.ClientEmail, key: key, origin: origin}, nil
}

// SaveURL erstellt den Link, über den der Teilnehmer den Pass in Google Wallet speichert.
// logoURL muss öffentlich über HTTPS erreichbar sein, sonst wird kein Logo verwendet.
func (g *Google) SaveURL(p Pass, logoURL string) (string, error) {
	classID := g.issuerID + ".midnightsports_teilnehmer"
	text := func(v string) map[string]any {
		return map[string]any{"defaultValue": map[string]string{"language": "de", "value": v}}
	}
	object := map[string]any{
		"id":                 g.issuerID + ".teilnehmer_" + p.UserID,
		"classId":            classID,
		"state":              "ACTIVE",
		"cardTitle":          text("Midnightsports"),
		"header":             text(p.name()),
		"subheader":          text("Teilnehmer"),
		"hexBackgroundColor": "#1b1f3b",
		"barcode": map[string]string{
			"type":  "QR_CODE",
			"value": p.Token,
		},
		"textModulesData": []any{
			map[string]string{"id": "geburtsdatum", "header": "Geburtsdatum", "body": p.birthDate()},
		},
	}
	if strings.HasPrefix(logoURL, "https://") {
		object["logo"] = map[string]any{"sourceUri": map[string]string{"uri": logoURL}}
	}

	claims := map[string]any{
		"iss":     g.email,
		"aud":     "google",
		"typ":     "savetowallet",
		"iat":     time.Now().Unix(),
		"origins": []string{g.origin},
		"payload": map[string]any{
			"genericClasses": []any{map[string]string{"id": classID}},
			"genericObjects": []any{object},
		},
	}
	jwt, err := signJWT(g.key, claims)
	if err != nil {
		return "", err
	}
	return googleSaveURL + jwt, nil
}

func signJWT(key *rsa.PrivateKey, claims any) (string, error) {
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := header + "." + enc.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + enc.EncodeToString(sig), nil
}
