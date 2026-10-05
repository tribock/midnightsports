package wallet

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/smallstep/pkcs7"
)

// Apple erstellt signierte .pkpass-Dateien für Apple Wallet.
type Apple struct {
	passTypeID string
	teamID     string
	cert       *x509.Certificate
	key        crypto.PrivateKey
	chain      []*x509.Certificate
}

// NewApple lädt das Pass-Zertifikat. Die PEM-Datei enthält das Pass Type ID Zertifikat,
// den privaten Schlüssel und das Apple WWDR-Zertifikat. Ohne Konfiguration wird nil zurückgegeben.
func NewApple(passTypeID, teamID, pemFile string) (*Apple, error) {
	if passTypeID == "" && teamID == "" && pemFile == "" {
		return nil, nil
	}
	if passTypeID == "" || teamID == "" || pemFile == "" {
		return nil, errors.New("Apple Wallet: APPLE_PASS_TYPE_ID, APPLE_TEAM_ID und APPLE_PASS_PEM_FILE sind alle nötig")
	}
	data, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, fmt.Errorf("Apple Wallet: %w", err)
	}
	cert, key, chain, err := parsePEMBundle(data)
	if err != nil {
		return nil, fmt.Errorf("Apple Wallet: %w", err)
	}
	return &Apple{passTypeID: passTypeID, teamID: teamID, cert: cert, key: key, chain: chain}, nil
}

// Build erstellt die .pkpass-Datei (ZIP mit pass.json, Bildern, manifest.json und Signatur).
func (a *Apple) Build(p Pass) ([]byte, error) {
	passJSON, err := json.Marshal(a.passJSON(p))
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{
		"pass.json":   passJSON,
		"icon.png":    IconPNG(29),
		"icon@2x.png": IconPNG(58),
		"icon@3x.png": IconPNG(87),
		"logo.png":    IconPNG(50),
		"logo@2x.png": IconPNG(100),
	}

	manifest := map[string]string{}
	for name, content := range files {
		sum := sha1.Sum(content)
		manifest[name] = hex.EncodeToString(sum[:])
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	signature, err := a.sign(manifestJSON)
	if err != nil {
		return nil, err
	}
	files["manifest.json"] = manifestJSON
	files["signature"] = signature

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *Apple) passJSON(p Pass) map[string]any {
	barcode := map[string]any{
		"format":          "PKBarcodeFormatQR",
		"message":         p.Token,
		"messageEncoding": "iso-8859-1",
	}
	return map[string]any{
		"formatVersion":      1,
		"passTypeIdentifier": a.passTypeID,
		"teamIdentifier":     a.teamID,
		"serialNumber":       p.UserID,
		"organizationName":   "Midnightsports",
		"description":        "Midnightsports Teilnehmerausweis",
		"logoText":           "Midnightsports",
		"backgroundColor":    "rgb(27, 31, 59)",
		"foregroundColor":    "rgb(255, 255, 255)",
		"labelColor":         "rgb(245, 200, 76)",
		"barcodes":           []any{barcode},
		"generic": map[string]any{
			"primaryFields": []any{
				map[string]string{"key": "name", "label": "Teilnehmer", "value": p.name()},
			},
			"secondaryFields": []any{
				map[string]string{"key": "birthDate", "label": "Geburtsdatum", "value": p.birthDate()},
			},
			"backFields": []any{
				map[string]string{"key": "info", "label": "Check-in",
					"value": "Zeige diesen QR-Code beim Eintreffen am Event dem Projektleiter."},
			},
		},
	}
}

// sign erstellt die detached PKCS#7-Signatur über manifest.json.
func (a *Apple) sign(manifest []byte) ([]byte, error) {
	sd, err := pkcs7.NewSignedData(manifest)
	if err != nil {
		return nil, err
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := sd.AddSignerChain(a.cert, a.key, a.chain, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, err
	}
	sd.Detach()
	return sd.Finish()
}

// parsePEMBundle liest Zertifikate und privaten Schlüssel. Das Zertifikat, das zum Schlüssel passt,
// signiert, alle anderen bilden die Kette.
func parsePEMBundle(data []byte) (*x509.Certificate, crypto.PrivateKey, []*x509.Certificate, error) {
	var certs []*x509.Certificate
	var key crypto.Signer
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, nil, err
			}
			certs = append(certs, c)
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			k, err := parsePrivateKey(block)
			if err != nil {
				return nil, nil, nil, err
			}
			key = k
		}
	}
	if key == nil {
		return nil, nil, nil, errors.New("kein privater Schlüssel in der PEM-Datei")
	}

	type publicKey interface{ Equal(crypto.PublicKey) bool }
	var signer *x509.Certificate
	var chain []*x509.Certificate
	for _, c := range certs {
		if pk, ok := c.PublicKey.(publicKey); ok && signer == nil && pk.Equal(key.Public()) {
			signer = c
		} else {
			chain = append(chain, c)
		}
	}
	if signer == nil {
		return nil, nil, nil, errors.New("kein Zertifikat passt zum privaten Schlüssel")
	}
	if len(chain) == 0 {
		return nil, nil, nil, errors.New("Apple WWDR-Zertifikat fehlt in der PEM-Datei")
	}
	return signer, key, chain, nil
}

func parsePrivateKey(block *pem.Block) (crypto.Signer, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	s, ok := k.(crypto.Signer)
	if !ok {
		return nil, errors.New("nicht unterstützter Schlüsseltyp")
	}
	return s, nil
}
