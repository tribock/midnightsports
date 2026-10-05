// Package qr erstellt und prüft die signierten Tokens der QR-Codes.
package qr

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

const prefix = "ms1"

var ErrInvalid = errors.New("ungültiger QR-Code")

var enc = base64.RawURLEncoding

// Sign erzeugt das Token für eine Benutzer-ID. Das Token ist pro Benutzer fest.
func Sign(key []byte, userID string) string {
	id := enc.EncodeToString([]byte(userID))
	return prefix + "." + id + "." + enc.EncodeToString(mac(key, id))
}

// Verify prüft das Token und gibt die Benutzer-ID zurück.
func Verify(key []byte, token string) (string, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != prefix {
		return "", ErrInvalid
	}
	sig, err := enc.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac(key, parts[1])) {
		return "", ErrInvalid
	}
	id, err := enc.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalid
	}
	return string(id), nil
}

func mac(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(prefix + "." + data))
	return h.Sum(nil)[:16]
}
