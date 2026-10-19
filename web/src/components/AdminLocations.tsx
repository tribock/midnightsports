import { useEffect, useState, type ChangeEvent, type FormEvent } from 'react'
import { api, errorMessage, type Location } from '../api'
import { formatAddress } from '../format'

const empty: Location = { id: 0, name: '', street: '', zip: '', city: '', isDefault: false }

/** Veranstaltungsorte als Liste, Bearbeiten direkt in der Zeile. Teil der Administration (nur Admins). */
export default function AdminLocations() {
  const [locations, setLocations] = useState<Location[]>([])
  // id des bearbeiteten Orts, 0 = neuer Ort
  const [editing, setEditing] = useState<number | null>(null)
  const [error, setError] = useState('')

  function load() {
    api
      .get<Location[]>('/locations')
      .then(setLocations)
      .catch((err) => setError(errorMessage(err)))
  }

  useEffect(load, [])

  function done() {
    setEditing(null)
    load()
  }

  return (
    <section className="stack">
      <div className="section-header">
        <h2>Veranstaltungsorte</h2>
        {editing !== 0 && (
          <button className="small" onClick={() => setEditing(0)}>
            + Neuer Ort
          </button>
        )}
      </div>
      {error && <p className="error">{error}</p>}
      <ul className="list">
        {locations.map((l) =>
          editing === l.id ? (
            <li key={l.id} className="editing">
              <LocationForm location={l} onDone={done} onCancel={() => setEditing(null)} />
            </li>
          ) : (
            <li key={l.id}>
              <span className="grow">
                <strong>{l.name}</strong>
                {l.isDefault && <span className="badge success">Standard</span>}
                <br />
                <span className="muted small">{formatAddress(l) || 'Keine Adresse'}</span>
              </span>
              <button className="small" onClick={() => setEditing(l.id)}>
                Bearbeiten
              </button>
            </li>
          ),
        )}
        {editing === 0 && (
          <li className="editing">
            <LocationForm location={empty} onDone={done} onCancel={() => setEditing(null)} />
          </li>
        )}
      </ul>
    </section>
  )
}

function LocationForm({ location, onDone, onCancel }: { location: Location; onDone: () => void; onCancel: () => void }) {
  const [form, setForm] = useState(location)
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      if (form.id) {
        await api.put(`/locations/${form.id}`, form)
      } else {
        await api.post('/locations', form)
      }
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  const set = (field: keyof Location) => (e: ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [field]: field === 'isDefault' ? e.target.checked : e.target.value })
  }

  return (
    <form className="stack" onSubmit={submit}>
      <div className="row">
        <label>
          Name
          <input required autoFocus value={form.name} onChange={set('name')} />
        </label>
        <label>
          Strasse und Nr.
          <input value={form.street} onChange={set('street')} />
        </label>
      </div>
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
      {error && <p className="error">{error}</p>}
      <div className="actions">
        <button type="button" className="small" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="small primary">
          {form.id ? 'Speichern' : 'Hinzufügen'}
        </button>
      </div>
    </form>
  )
}
