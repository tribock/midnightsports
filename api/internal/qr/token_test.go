package qr

import "testing"

func TestSignVerify(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	id := "5b0c3f4e-8d2a-4b1e-9f3a-1c2d3e4f5a6b"

	token := Sign(key, id)
	got, err := Verify(key, token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != id {
		t.Fatalf("got %q, want %q", got, id)
	}
	if Sign(key, id) != token {
		t.Fatal("Token muss pro Benutzer stabil sein")
	}
}

func TestVerifyRejects(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	other := []byte("fedcba9876543210fedcba9876543210")
	token := Sign(key, "user-1")
	forged := Sign(other, "user-2")

	for name, tok := range map[string]string{
		"leer":            "",
		"falscher Key":    Sign(other, "user-1"),
		"fremde Signatur": token[:len(token)-22] + forged[len(forged)-22:],
		"ohne Prefix":     token[4:],
		"Benutzer-ID":     "user-1",
	} {
		if _, err := Verify(key, tok); err == nil {
			t.Errorf("%s: Token wurde akzeptiert", name)
		}
	}
}
