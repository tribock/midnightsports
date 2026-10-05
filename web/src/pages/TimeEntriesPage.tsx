import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage, type Event, type TimeEntry } from '../api'
import { useAuth } from '../auth'
import { formatShortDate, todayISO } from '../format'

const emptyForm = { date: todayISO(), hours: '', description: '', eventId: '' }

export default function TimeEntriesPage() {
  const { user } = useAuth()
  const admin = user?.role === 'admin'
  const [all, setAll] = useState(false)
  const [entries, setEntries] = useState<TimeEntry[]>([])
  const [events, setEvents] = useState<Event[]>([])
  const [form, setForm] = useState(emptyForm)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<TimeEntry[]>(`/time-entries${all ? '?all=true' : ''}`)
      .then(setEntries)
      .catch((err) => setError(errorMessage(err)))
  }, [all])

  useEffect(load, [load])
  useEffect(() => {
    Promise.all([api.get<Event[]>('/events?past=true'), api.get<Event[]>('/events')]).then(([past, upcoming]) =>
      setEvents([...upcoming.reverse(), ...past]),
    )
  }, [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.post('/time-entries', {
        date: form.date,
        hours: Number(form.hours.replace(',', '.')),
        description: form.description,
        eventId: form.eventId ? Number(form.eventId) : null,
      })
      setForm({ ...emptyForm, date: form.date })
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove(entry: TimeEntry) {
    if (!confirm('Eintrag löschen?')) return
    await api.del(`/time-entries/${entry.id}`).catch((err) => setError(errorMessage(err)))
    load()
  }

  const total = entries.reduce((sum, e) => sum + e.hours, 0)
  return (
    <div className="stack">
      <div className="page-header">
        <h1>Zeiterfassung</h1>
        {admin && (
          <div className="actions">
            <button onClick={() => setAll(!all)}>{all ? 'Nur meine' : 'Alle anzeigen'}</button>
            <a className="button" href="/api/time-entries/export.csv">
              CSV exportieren
            </a>
          </div>
        )}
      </div>

      <form className="card stack" onSubmit={submit}>
        <div className="row">
          <label>
            Datum
            <input type="date" required value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} />
          </label>
          <label>
            Stunden
            <input
              inputMode="decimal"
              required
              pattern="[0-9]+([.,][0-9]{1,2})?"
              placeholder="z.B. 3.5"
              value={form.hours}
              onChange={(e) => setForm({ ...form, hours: e.target.value })}
            />
          </label>
        </div>
        <label>
          Event (optional)
          <select value={form.eventId} onChange={(e) => setForm({ ...form, eventId: e.target.value })}>
            <option value="">–</option>
            {events.map((ev) => (
              <option key={ev.id} value={ev.id}>
                {formatShortDate(ev.date)} · {ev.location.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Beschreibung
          <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </label>
        {error && <p className="error">{error}</p>}
        <button type="submit" className="primary">
          Erfassen
        </button>
      </form>

      <table className="table">
        <thead>
          <tr>
            <th>Datum</th>
            {all && <th>Name</th>}
            <th className="num">Std.</th>
            <th>Beschreibung</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr key={e.id}>
              <td>{formatShortDate(e.date)}</td>
              {all && <td>{e.userName}</td>}
              <td className="num">{e.hours.toFixed(2)}</td>
              <td>
                {e.description}
                {e.eventDate && <span className="muted small"> · Event {formatShortDate(e.eventDate)}</span>}
              </td>
              <td>
                <button className="link small" onClick={() => remove(e)} aria-label="Löschen">
                  ✕
                </button>
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td>Total</td>
            {all && <td />}
            <td className="num">{total.toFixed(2)}</td>
            <td colSpan={2} />
          </tr>
        </tfoot>
      </table>
    </div>
  )
}
