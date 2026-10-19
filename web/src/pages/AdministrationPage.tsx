import { useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage, type EmergencyAccess, type Func, type Role } from '../api'
import { useAuth } from '../auth'
import AdminLocations from '../components/AdminLocations'
import { parseDecimal, roleLabels, salaryLabel } from '../format'

/** Administration (nur Admins): Funktionen mit Lohn pro Einsatz, Zugriff auf Notfallkontakte und Veranstaltungsorte. */
export default function AdministrationPage() {
  const [functions, setFunctions] = useState<Func[]>([])
  const [error, setError] = useState('')

  function load() {
    api
      .get<Func[]>('/functions')
      .then(setFunctions)
      .catch((err) => setError(errorMessage(err)))
  }

  useEffect(load, [])

  return (
    <div className="stack">
      <h1>Administration</h1>
      {error && <p className="error">{error}</p>}
      <Functions functions={functions} onChanged={load} />
      <EmergencyAccessSettings functions={functions} />
      <AdminLocations />
    </div>
  )
}

/** Funktionen als Liste, Bearbeiten direkt in der Zeile. */
function Functions({ functions, onChanged }: { functions: Func[]; onChanged: () => void }) {
  // id der bearbeiteten Funktion, 0 = neue Funktion
  const [editing, setEditing] = useState<number | null>(null)

  function done() {
    setEditing(null)
    onChanged()
  }

  return (
    <section className="stack">
      <div className="section-header">
        <h2>Funktionen</h2>
        {editing !== 0 && (
          <button className="small" onClick={() => setEditing(0)}>
            + Neue Funktion
          </button>
        )}
      </div>
      <p className="muted small">
        Lohn pro Einsatz an einem Event. Ohne Lohn (z.B. PL) wird über die Zeiterfassung abgerechnet.
      </p>
      <ul className="list">
        {functions.map((f) =>
          editing === f.id ? (
            <li key={f.id} className="editing">
              <FunctionForm fn={f} onDone={done} onCancel={() => setEditing(null)} />
            </li>
          ) : (
            <li key={f.id}>
              <strong className="grow">{f.name}</strong>
              <span className={f.salary === null ? 'muted small' : 'num'}>{salaryLabel(f.salary)}</span>
              <button className="small" onClick={() => setEditing(f.id)}>
                Bearbeiten
              </button>
            </li>
          ),
        )}
        {editing === 0 && (
          <li className="editing">
            <FunctionForm fn={{ id: 0, name: '', salary: null }} onDone={done} onCancel={() => setEditing(null)} />
          </li>
        )}
        {functions.length === 0 && editing !== 0 && <li className="muted">Noch keine Funktionen erfasst.</li>}
      </ul>
    </section>
  )
}

function FunctionForm({ fn, onDone, onCancel }: { fn: Func; onDone: () => void; onCancel: () => void }) {
  const [name, setName] = useState(fn.name)
  const [salary, setSalary] = useState(fn.salary === null ? '' : String(fn.salary))
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    const body = { name, salary: parseDecimal(salary) }
    try {
      if (fn.id) {
        await api.put(`/functions/${fn.id}`, body)
      } else {
        await api.post('/functions', body)
      }
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove() {
    if (!confirm(`Funktion ${fn.name} löschen?`)) return
    try {
      await api.del(`/functions/${fn.id}`)
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="stack" onSubmit={submit}>
      <div className="row">
        <label>
          Name
          <input required autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder="z.B. JC" />
        </label>
        <label>
          Lohn pro Einsatz (CHF)
          <input
            inputMode="decimal"
            pattern="[0-9]+([.,][0-9]{1,2})?"
            value={salary}
            onChange={(e) => setSalary(e.target.value)}
            placeholder="leer = über Zeiterfassung"
          />
        </label>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="actions">
        {fn.id !== 0 && (
          <button type="button" className="small danger" onClick={remove}>
            Löschen
          </button>
        )}
        <span className="grow" />
        <button type="button" className="small" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="small primary">
          {fn.id ? 'Speichern' : 'Hinzufügen'}
        </button>
      </div>
    </form>
  )
}

/** Wählbare Rollen: Admins haben immer Zugriff auf Notfallkontakte, Teilnehmer nie */
const accessRoles: Role[] = ['projektleiter', 'mitarbeiter']

/** Rollen und Funktionen, die Notfallkontakte einsehen dürfen. Jede Änderung wird sofort gespeichert. */
function EmergencyAccessSettings({ functions }: { functions: Func[] }) {
  const { refresh } = useAuth()
  const [access, setAccess] = useState<EmergencyAccess | null>(null)
  const [error, setError] = useState('')

  // Neu laden, wenn Funktionen ändern (gelöschte Funktionen verlieren ihren Zugriff)
  useEffect(() => {
    api
      .get<EmergencyAccess>('/emergency-access')
      .then(setAccess)
      .catch((err) => setError(errorMessage(err)))
  }, [functions])

  function toggle<T>(list: T[], value: T): T[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
  }

  async function save(next: EmergencyAccess) {
    const previous = access
    setAccess(next)
    setError('')
    try {
      setAccess(await api.put<EmergencyAccess>('/emergency-access', next))
      // Der eigene Zugriff kann sich geändert haben
      await refresh()
    } catch (err) {
      setAccess(previous)
      setError(errorMessage(err))
    }
  }

  if (!access) return error ? <p className="error">{error}</p> : null
  return (
    <section className="stack">
      <div className="section-header">
        <h2>Zugriff auf Notfallkontakte</h2>
      </div>
      <p className="muted small">
        Gewählte Rollen und Funktionen dürfen die Notfallkontakte aller Benutzer einsehen. Admins immer, Teilnehmer nie.
      </p>
      <div className="card access-grid">
        <span className="muted small">Rollen</span>
        <div className="chips">
          <button className="chip active" disabled title="Admins haben immer Zugriff">
            {roleLabels.admin}
          </button>
          {accessRoles.map((role) => (
            <button
              key={role}
              className={access.roles.includes(role) ? 'chip active' : 'chip'}
              aria-pressed={access.roles.includes(role)}
              onClick={() => save({ ...access, roles: toggle(access.roles, role) })}
            >
              {roleLabels[role]}
            </button>
          ))}
        </div>
        <span className="muted small">Funktionen</span>
        <div className="chips">
          {functions.length === 0 && <span className="muted small">Noch keine Funktionen erfasst.</span>}
          {functions.map((f) => (
            <button
              key={f.id}
              className={access.functionIds.includes(f.id) ? 'chip active' : 'chip'}
              aria-pressed={access.functionIds.includes(f.id)}
              onClick={() => save({ ...access, functionIds: toggle(access.functionIds, f.id) })}
            >
              {f.name}
            </button>
          ))}
        </div>
      </div>
      {error && <p className="error">{error}</p>}
    </section>
  )
}
