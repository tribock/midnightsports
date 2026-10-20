package wallet

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"image/png"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
)

var testPass = Pass{
	UserID:    "f871ab10-f6ae-4353-8539-ff9a76b56225",
	Token:     "ms1.abc.def",
	FirstName: "Lena",
	LastName:  "Muster",
	BirthDate: "2010-03-14",
}

func TestNotConfigured(t *testing.T) {
	if a, err := NewApple("", "", ""); a != nil || err != nil {
		t.Errorf("Apple: %v, %v", a, err)
	}
	if g, err := NewGoogle("", "", ""); g != nil || err != nil {
		t.Errorf("Google: %v, %v", g, err)
	}
	if _, err := NewApple("pass.ch.test", "", ""); err == nil {
		t.Error("Apple: unvollständige Konfiguration muss fehlschlagen")
	}
}

func TestApplePass(t *testing.T) {
	pemFile := writeTestBundle(t)
	a, err := NewApple("pass.ch.midnightsports", "ABCDE12345", pemFile)
	if err != nil {
		t.Fatal(err)
	}
	pkpass, err := a.Build(testPass)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(pkpass), int64(len(pkpass)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		r, _ := f.Open()
		files[f.Name], _ = io.ReadAll(r)
		r.Close()
	}
	for _, name := range []string{"pass.json", "manifest.json", "signature", "icon.png", "icon@2x.png"} {
		if files[name] == nil {
			t.Errorf("%s fehlt", name)
		}
	}

	var pass map[string]any
	json.Unmarshal(files["pass.json"], &pass)
	barcode := pass["barcodes"].([]any)[0].(map[string]any)
	if barcode["message"] != testPass.Token || pass["serialNumber"] != testPass.UserID {
		t.Errorf("pass.json: %s", files["pass.json"])
	}

	var manifest map[string]string
	json.Unmarshal(files["manifest.json"], &manifest)
	for name, hash := range manifest {
		sum := sha1.Sum(files[name])
		if hex.EncodeToString(sum[:]) != hash {
			t.Errorf("manifest: falscher Hash für %s", name)
		}
	}

	p7, err := pkcs7.Parse(files["signature"])
	if err != nil {
		t.Fatal(err)
	}
	p7.Content = files["manifest.json"]
	if err := p7.Verify(); err != nil {
		t.Errorf("Signatur ungültig: %v", err)
	}
}

func TestGoogleSaveURL(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	sa, _ := json.Marshal(map[string]string{
		"client_email": "wallet@test.iam.gserviceaccount.com",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	keyFile := filepath.Join(t.TempDir(), "sa.json")
	os.WriteFile(keyFile, sa, 0o600)

	g, err := NewGoogle("3388000000012345678", keyFile, "https://midnightsports.example")
	if err != nil {
		t.Fatal(err)
	}
	url, err := g.SaveURL(testPass, "https://midnightsports.example/api/wallet/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	jwt, ok := strings.CutPrefix(url, googleSaveURL)
	if !ok {
		t.Fatalf("falsche URL: %s", url)
	}

	parts := strings.Split(jwt, ".")
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("JWT-Signatur ungültig: %v", err)
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	for _, want := range []string{`"typ":"savetowallet"`, `"value":"ms1.abc.def"`, `.teilnehmer_` + testPass.UserID, `"logo"`} {
		if !strings.Contains(string(payload), want) {
			t.Errorf("JWT enthält %s nicht: %s", want, payload)
		}
	}
}

// writeTestBundle erstellt eine PEM-Datei wie aus dem Apple-Zertifikat: Pass-Zertifikat, Schlüssel, WWDR.
func writeTestBundle(t *testing.T) string {
	t.Helper()
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test WWDR"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	ca, _ := x509.ParseCertificate(caDER)

	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Pass Type ID: pass.ch.midnightsports"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, caKey)

	var buf bytes.Buffer
	pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	pem.Encode(&buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	path := filepath.Join(t.TempDir(), "pass.pem")
	os.WriteFile(path, buf.Bytes(), 0o600)
	return path
}

// Die Logos werden mit `make logos` erzeugt. Apple verlangt genaue Grössen, sonst fehlt das Bild auf dem Pass
func TestAssets(t *testing.T) {
	tests := map[string][2]int{
		"icon-29.png":  {29, 29},
		"icon-58.png":  {58, 58},
		"icon-87.png":  {87, 87},
		"icon-660.png": {660, 660},
		"logo.png":     {160, 0},
		"logo@2x.png":  {320, 0},
	}
	for name, want := range tests {
		cfg, err := png.DecodeConfig(bytes.NewReader(asset(name)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if cfg.Width != want[0] || (want[1] != 0 && cfg.Height != want[1]) {
			t.Errorf("%s: %dx%d, want %dx%d", name, cfg.Width, cfg.Height, want[0], want[1])
		}
		// Apple: Logo höchstens 160 x 50 Punkte
		if strings.HasPrefix(name, "logo") && cfg.Height*160 > cfg.Width*50 {
			t.Errorf("%s: %dx%d zu hoch für Apple Wallet", name, cfg.Width, cfg.Height)
		}
	}
}
