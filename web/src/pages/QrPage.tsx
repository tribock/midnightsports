import { useEffect, useState } from 'react'
import QRCode from 'react-qr-code'
import { api, errorMessage } from '../api'
import { useAuth } from '../auth'

export default function QrPage() {
  const { user } = useAuth()
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [wallets, setWallets] = useState({ appleWallet: false, googleWallet: false })

  useEffect(() => {
    api
      .get<{ token: string }>('/me/qr')
      .then((r) => setToken(r.token))
      .catch((err) => setError(errorMessage(err)))
    api.get<typeof wallets>('/config').then(setWallets)
  }, [])

  return (
    <div className="narrow center">
      <h1>Mein QR-Code</h1>
      <p className="muted">Zeige diesen Code beim Eintreffen am Event dem Projektleiter.</p>
      {error && <p className="error">{error}</p>}
      {token && (
        <div className="qr">
          <QRCode value={token} size={260} />
        </div>
      )}
      <p className="qr-name">
        {user?.firstName} {user?.lastName}
      </p>
      {token && (wallets.appleWallet || wallets.googleWallet) && (
        <div className="wallet-buttons">
          {wallets.appleWallet && (
            <a className="button wallet" href="/api/me/wallet/apple">
              Zu Apple Wallet hinzufügen
            </a>
          )}
          {wallets.googleWallet && (
            <a className="button wallet" href="/api/me/wallet/google">
              Zu Google Wallet hinzufügen
            </a>
          )}
        </div>
      )}
    </div>
  )
}
