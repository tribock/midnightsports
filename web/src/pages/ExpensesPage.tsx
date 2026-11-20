import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { errorMessage, type Event, type ExpenseEntry } from '../api'
import { isProjectLead } from '../auth'
import { formatShortDate, parseDecimal, todayISO } from '../format'
import { useProject } from '../project'

const emptyForm = { date: todayISO(), amount: '', reason: '', eventId: '' }

export default function ExpensesPage() {
  const { api, member } = useProject()
  const projectLead = isProjectLead(member)
  const [all, setAll] = useState(false)
  const [entries, setEntries] = useState<ExpenseEntry[]>([])
  const [events, setEvents] = useState<Event[]>([])
  const [form, setForm] = useState(emptyForm)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<ExpenseEntry[]>(`/expenses${all ? '?all=true' : ''}`)
      .then(setEntries)
      .catch((err) => setError(errorMessage(err)))
  }, [api, all])

  useEffect(load, [load])
  useEffect(() => {
    Promise.all([api.get<Event[]>('/events?past=true'), api.get<Event[]>('/events')]).then(([past, upcoming]) =>
      setEvents([...upcoming.reverse(), ...past]),
    )
  }, [api])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.post('/expenses', {
        date: form.date,
        amount: parseDecimal(form.amount),
        reason: form.reason,
        eventId: form.eventId ? Number(form.eventId) : null,
      })
      setForm({ ...emptyForm, date: form.date })
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove(entry: ExpenseEntry) {
    if (!confirm('Eintrag löschen?')) return
    await api.del(`/expenses/${entry.id}`).catch((err) => setError(errorMessage(err)))
    load()
  }

  const total = entries.reduce((sum, e) => sum + e.amount, 0)
  return (
    <div className="stack">
      <div className="page-header">
        <h1>Spesen</h1>
        {projectLead && (
          <div className="actions">
            <button onClick={() => setAll(!all)}>{all ? 'Nur meine' : 'Alle anzeigen'}</button>
            <a className="button" href={`/api/p/${member.project.slug}/expenses/export.csv`}>
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
            Betrag (CHF)
            <input
              inputMode="decimal"
              required
              pattern="[0-9]+([.,][0-9]{1,2})?"
              placeholder="z.B. 3.50"
              value={form.amount}
              onChange={(e) => setForm({ ...form, amount: e.target.value })}
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
          Begründung
          <input value={form.reason
          } onChange={(e) => setForm({ ...form, reason: e.target.value })} />
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
            <th className="num">Betrag (CHF)</th>
            <th>Begründung</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr key={e.id}>
              <td>{formatShortDate(e.date)}</td>
              {all && <td>{e.userName}</td>}
              <td className="num">{e.amount.toFixed(2)}</td>
              <td>
                {e.reason}
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
