import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, errorMessage, type Event, type EventInput, type Location, type UserSummary } from '../api'
import { useAuth } from '../auth'
import { todayISO } from '../format'

export default function EventFormPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()
  const [locations, setLocations] = useState<Location[]>([])
  const [leaders, setLeaders] = useState<UserSummary[]>([])
  const [form, setForm] = useState<EventInput>({
    date: todayISO(),
    startTime: '21:00',
    endTime: '00:00',
    locationId: 0,
    leaderIds: [user!.id],
  })
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([
      api.get<Location[]>('/locations'),
      api.get<UserSummary[]>('/users?role=projektleiter'),
      id ? api.get<Event>(`/events/${id}`) : Promise.resolve(null),
    ])
      .then(([locs, users, event]) => {
        setLocations(locs)
        setLeaders(users)
        if (event) {
          setForm({
            date: event.date,
            startTime: event.startTime,
            endTime: event.endTime,
            locationId: event.location.id,
            leaderIds: event.leaders.map((l) => l.id),
          })
        } else {
          setForm((f) => ({ ...f, locationId: locs.find((l) => l.isDefault)?.id ?? locs[0]?.id ?? 0 }))
        }
      })
      .catch((err) => setError(errorMessage(err)))
  }, [id])

  function toggleLeader(leaderId: string) {
    const ids = form.leaderIds.includes(leaderId)
      ? form.leaderIds.filter((x) => x !== leaderId)
      : [...form.leaderIds, leaderId]
    setForm({ ...form, leaderIds: ids })
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const saved = id ? await api.put<Event>(`/events/${id}`, form) : await api.post<Event>('/events', form)
      navigate(`/events/${saved.id}`)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="narrow">
      <h1>{id ? 'Event bearbeiten' : 'Neuer Event'}</h1>
      <form className="card stack" onSubmit={submit}>
        <label>
          Datum
          <input type="date" required value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} />
        </label>
        <div className="row">
          <label>
            Von
            <input
              type="time"
              required
              value={form.startTime}
              onChange={(e) => setForm({ ...form, startTime: e.target.value })}
            />
          </label>
          <label>
            Bis
            <input
              type="time"
              required
              value={form.endTime}
              onChange={(e) => setForm({ ...form, endTime: e.target.value })}
            />
          </label>
        </div>
        <p className="muted small">Liegt das Ende vor dem Start, endet der Event am nächsten Tag.</p>
        <label>
          Veranstaltungsort
          <select
            value={form.locationId}
            onChange={(e) => setForm({ ...form, locationId: Number(e.target.value) })}
          >
            {locations.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </select>
        </label>
        <fieldset>
          <legend>Projektleiter</legend>
          {leaders.map((l) => (
            <label key={l.id} className="check">
              <input type="checkbox" checked={form.leaderIds.includes(l.id)} onChange={() => toggleLeader(l.id)} />
              {l.firstName} {l.lastName}
            </label>
          ))}
        </fieldset>
        {error && <p className="error">{error}</p>}
        <div className="actions">
          <button type="button" onClick={() => navigate(-1)}>
            Abbrechen
          </button>
          <button type="submit" className="primary">
            Speichern
          </button>
        </div>
      </form>
    </div>
  )
}
