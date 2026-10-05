import { useEffect, useState, type FormEvent } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { api, errorMessage } from '../api'
import { useAuth } from '../auth'

const errors: Record<string, string> = {
  state: 'Die Anmeldung ist abgelaufen. Bitte erneut versuchen.',
  google: 'Die Anmeldung bei Google ist fehlgeschlagen.',
  email: 'Die Email-Adresse des Google Accounts ist nicht bestätigt.',
  server: 'Die Anmeldung ist fehlgeschlagen. Bitte später erneut versuchen.',
}

export default function LoginPage() {
  const { user, loading, refresh } = useAuth()
  const [params] = useSearchParams()
  const [devLogin, setDevLogin] = useState(false)
  const [email, setEmail] = useState('')
  const [error, setError] = useState(errors[params.get('error') ?? ''] ?? '')

  useEffect(() => {
    api.get<{ devLogin: boolean }>('/config').then((c) => setDevLogin(c.devLogin))
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
      <img src="/icon.svg" alt="" width={96} height={96} />
      <h1>Midnightsports</h1>
      <p className="muted">Events in den Turnhallen Guntershausen, Aadorf und Ettenhausen</p>
      {error && <p className="error">{error}</p>}
      <a className="button primary block" href="/api/auth/login">
        Mit Google anmelden
      </a>

      {devLogin && (
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
