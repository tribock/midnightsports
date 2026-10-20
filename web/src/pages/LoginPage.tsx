import { useEffect, useState, type FormEvent } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { api, errorMessage } from '../api'
import { useAuth } from '../auth'

const errors: Record<string, string> = {
  state: 'Die Anmeldung ist abgelaufen. Bitte erneut versuchen.',
  google: 'Die Anmeldung bei Google ist fehlgeschlagen.',
  apple: 'Die Anmeldung bei Apple ist fehlgeschlagen.',
  email: 'Die Email-Adresse des Accounts ist nicht bestätigt.',
  server: 'Die Anmeldung ist fehlgeschlagen. Bitte später erneut versuchen.',
}

export default function LoginPage() {
  const { user, loading, refresh } = useAuth()
  const [params] = useSearchParams()
  const [config, setConfig] = useState({ devLogin: false, appleLogin: false })
  const [email, setEmail] = useState('')
  const [error, setError] = useState(errors[params.get('error') ?? ''] ?? '')

  useEffect(() => {
    api.get<typeof config>('/config').then(setConfig)
  }, [])

  if (loading) return null
  if (user) return <Navigate to="/" replace />

  async function submitDevLogin(e: FormEvent) {
    e.preventDefault()
    try {
      await api.post('/auth/dev-login', { email })
      await refresh()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <main className="container narrow login">
      <h1 className="login-logo">
        {/* Im Dark Mode die weisse Variante, das blaue Logo wäre auf dunklem Grund kaum lesbar */}
        <picture>
          <source srcSet="/logo-white.svg" media="(prefers-color-scheme: dark)" />
          <img src="/logo.svg" alt="Midnightsports" width={280} height={50} />
        </picture>
      </h1>
      <p className="muted">Events in den Turnhallen Guntershausen, Aadorf und Ettenhausen</p>
      {error && <p className="error">{error}</p>}
      <a className="button primary block" href="/api/auth/login">
        Mit Google anmelden
      </a>
      {config.appleLogin && (
        <a className="button apple block" href="/api/auth/apple/login">
          Mit Apple anmelden
        </a>
      )}

      {config.devLogin && (
        <form className="card stack" onSubmit={submitDevLogin}>
          <strong>Entwicklung: Anmeldung ohne Google</strong>
          <label>
            Email
            <input type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          </label>
          <button type="submit">Anmelden</button>
        </form>
      )}
    </main>
  )
}
