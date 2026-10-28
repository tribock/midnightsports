import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, errorMessage, type Me, type User } from '../api'
import { afterLogin, useAuth } from '../auth'
import ProfileFields, { type ProfileData } from '../components/ProfileFields'

export default function ProfilePage() {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState<ProfileData>(user!)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const firstTime = !user!.profileComplete

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const updated = await api.put<User>('/me', form)
      setUser({ ...(user as Me), ...updated })
      setSaved(true)
      // Nach dem ersten Ausfüllen weiter zum Projekt-Link oder zur Startseite
      if (firstTime) navigate(afterLogin(true) ?? '/')
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function deleteAccount() {
    if (!confirm('Konto und alle persönlichen Daten endgültig löschen?')) return
    try {
      await api.del('/me')
      setUser(null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="narrow">
      <h1>Profil</h1>
      {firstTime && <p className="notice">Bitte fülle dein Profil aus, bevor du dich für Projekte und Events anmelden kannst.</p>}
      <p className="muted small">Das Profil gilt für alle Projekte, bei denen du angemeldet bist.</p>

      <form className="stack" onSubmit={submit}>
        <ProfileFields
          value={form}
          email="readonly"
          onChange={(v) => {
            setForm(v)
            setSaved(false)
          }}
        />
        {error && <p className="error">{error}</p>}
        {saved && <p className="success">Gespeichert</p>}
        <button type="submit" className="primary">
          Speichern
        </button>
      </form>

      <section className="danger-zone">
        <h2>Konto löschen</h2>
        <p className="muted">
          Alle persönlichen Daten und Anmeldungen bei Projekten werden gelöscht. Besuche an Events bleiben anonym für
          die Statistik erhalten.
        </p>
        <button className="danger" onClick={deleteAccount}>
          Konto löschen
        </button>
      </section>
    </div>
  )
}
