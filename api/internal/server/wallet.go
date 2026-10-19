package server

import (
	"net/http"
	"sync"

	"github.com/tribock/midnightsports/api/internal/qr"
	"github.com/tribock/midnightsports/api/internal/wallet"
)

func (s *Server) walletPass(u *User) (wallet.Pass, error) {
	if !u.ProfileComplete {
		return wallet.Pass{}, errIncomplete
	}
	return wallet.Pass{
		UserID:    u.ID,
		Token:     qr.Sign(s.cfg.QRSigningKey, u.ID),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		BirthDate: u.BirthDate,
	}, nil
}

// getAppleWalletPass liefert den Pass als .pkpass-Datei. iOS öffnet sie direkt in Wallet.
func (s *Server) getAppleWalletPass(w http.ResponseWriter, r *http.Request, u *User) error {
	if s.appleWallet == nil {
		return errNotFound
	}
	pass, err := s.walletPass(u)
	if err != nil {
		return err
	}
	pkpass, err := s.appleWallet.Build(pass)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/vnd.apple.pkpass")
	w.Header().Set("Content-Disposition", `attachment; filename="midnightsports.pkpass"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(pkpass)
	return nil
}

// getGoogleWalletLink leitet auf die Google-Seite zum Speichern des Passes weiter.
func (s *Server) getGoogleWalletLink(w http.ResponseWriter, r *http.Request, u *User) error {
	if s.googleWallet == nil {
		return errNotFound
	}
	pass, err := s.walletPass(u)
	if err != nil {
		return err
	}
	url, err := s.googleWallet.SaveURL(pass, s.cfg.BaseURL()+"/api/wallet/logo.png")
	if err != nil {
		return err
	}
	http.Redirect(w, r, url, http.StatusFound)
	return nil
}

var walletLogo = sync.OnceValue(func() []byte { return wallet.IconPNG(660) })

// getWalletLogo liefert das Logo, das Google Wallet öffentlich abruft.
func (s *Server) getWalletLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(walletLogo())
}
