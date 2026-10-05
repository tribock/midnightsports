import { useState, type FormEvent, type InputHTMLAttributes } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, errorMessage, type User } from '../api'
import { useAuth } from '../auth'
import { todayISO } from '../format'

type Field = keyof Pick<
  User,
  | 'firstName'
  | 'lastName'
  | 'birthDate'
  | 'street'
  | 'zip'
  | 'city'
  | 'emergencyName'
  | 'emergencyPhone'
  | 'emergencyRelation'
>

export default function ProfilePage() {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState<User>(user!)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const firstTime = !user!.profileComplete

  function input(field: Field, label: string, props: InputHTMLAttributes<HTMLInputElement> = {}) {
    return (
      <label>
        {label}
        <input
          required
          value={form[field]}
          onChange={(e) => {
            setForm({ ...form, [field]: e.target.value })
            setSaved(false)
          }}
          {...props}
        />
      </label>
    )
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const updated = await api.put<User>('/me', form)
      setUser(updated)
      setSaved(true)
      if (firstTime) navigate('/events')
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
      {firstTime && <p className="notice">Bitte fülle dein Profil aus, bevor du dich für Events anmelden kannst.</p>}

      <form className="stack" onSubmit={submit}>
        <fieldset className="card stack">
          <legend>Person</legend>
          <label>
            Email
            <input value={form.email} disabled />
          </label>
          <div className="row">
            {input('firstName', 'Vorname', { autoComplete: 'given-name' })}
            {input('lastName', 'Name', { autoComplete: 'family-name' })}
          </div>
          {input('birthDate', 'Geburtsdatum', { type: 'date', max: todayISO(), autoComplete: 'bday' })}
        </fieldset>

        <fieldset className="card stack">
          <legend>Adresse</legend>
          {input('street', 'Strasse und Nr.', { autoComplete: 'street-address' })}
          <div className="row">
            {input('zip', 'PLZ', { inputMode: 'numeric', autoComplete: 'postal-code', className: 'zip' })}
            {input('city', 'Ort', { autoComplete: 'address-level2' })}
          </div>
        </fieldset>

        <fieldset className="card stack">
          <legend>Notfallkontakt</legend>
          {input('emergencyName', 'Name')}
          <div className="row">
            {input('emergencyPhone', 'Telefon', { type: 'tel' })}
            {input('emergencyRelation', 'Beziehung', { placeholder: 'z.B. Mutter' })}
          </div>
        </fieldset>

        {error && <p className="error">{error}</p>}
        {saved && <p className="success">Gespeichert</p>}
        <button type="submit" className="primary">
          Speichern
        </button>
      </form>

      <section className="danger-zone">
        <h2>Konto löschen</h2>
        <p className="muted">
          Alle persönlichen Daten werden gelöscht. Besuche an Events bleiben anonym für die Statistik erhalten.
        </p>
        <button className="danger" onClick={deleteAccount}>
          Konto löschen
        </button>
      </section>
    </div>
  )
}
