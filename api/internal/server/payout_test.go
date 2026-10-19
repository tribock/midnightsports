package server

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseSignature(t *testing.T) {
	png := base64.StdEncoding.EncodeToString(append(pngMagic, "rest"...))

	tests := []struct {
		name  string
		input string
		ok    bool
	}{
		{"base64", png, true},
		{"data-URL", "data:image/png;base64," + png, true},
		{"kein PNG", base64.StdEncoding.EncodeToString([]byte("GIF89a")), false},
		{"kein base64", "%%%", false},
		{"leer", "", false},
		{"zu gross", base64.StdEncoding.EncodeToString(append(pngMagic, strings.Repeat("x", maxSignatureSize)...)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSignature(tt.input)
			if (err == nil) != tt.ok {
				t.Errorf("parseSignature() error = %v, want ok = %v", err, tt.ok)
			}
		})
	}
}
