import { useEffect, useState, type ChangeEvent, type FormEvent } from 'react'
import { api, errorMessage, type Location } from '../api'

const empty: Location = { id: 0, name: '', street: '', zip: '', city: '', isDefault: false }

export default function AdminLocationsPage() {
  const [locations, setLocations] = useState<Location[]>([])
  const [error, setError] = useState('')

  function load() {
    api
      .get<Location[]>('/locations')
      .then(setLocations)
      .catch((err) => setError(errorMessage(err)))
  }

  useEffect(load, [])

  return (
    <div className="stack">
      <h1>Veranstaltungsorte</h1>
      {error && <p className="error">{error}</p>}
      {locations.map((l) => (
        <LocationForm key={l.id} location={l} onSaved={load} />
      ))}
      <h2>Neuer Ort</h2>
      <LocationForm location={empty} onSaved={load} />
    </div>
  )
}

function LocationForm({ location, onSaved }: { location: Location; onSaved: () => void }) {
  const [form, setForm] = useState(location)
  const [status, setStatus] = useState('')

  useEffect(() => setForm(location), [location])

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      if (form.id) {
        await api.put(`/locations/${form.id}`, form)
      } else {
        await api.post('/locations', form)
        setForm(empty)
      }
      setStatus('Gespeichert')
      onSaved()
    } catch (err) {
      setStatus(errorMessage(err))
    }
  }

  const set = (field: keyof Location) => (e: ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [field]: field === 'isDefault' ? e.target.checked : e.target.value })
    setStatus('')
  }

  return (
    <form className="card stack" onSubmit={submit}>
      <label>
        Name
        <input required value={form.name} onChange={set('name')} />
      </label>
      <label>
        Strasse und Nr.
        <input value={form.street} onChange={set('street')} />
      </label>
      <div className="row">
        <label className="zip">
          PLZ
          <input value={form.zip} onChange={set('zip')} />
        </label>
        <label>
          Ort
          <input value={form.city} onChange={set('city')} />
        </label>
      </div>
      <label className="check">
        <input type="checkbox" checked={form.isDefault} disabled={location.isDefault} onChange={set('isDefault')} />
        Standardort für neue Events
      </label>
      <div className="actions">
        {status && <span className="muted small">{status}</span>}
        <button type="submit" className="primary">
          {form.id ? 'Speichern' : 'Hinzufügen'}
        </button>
      </div>
    </form>
  )
}
