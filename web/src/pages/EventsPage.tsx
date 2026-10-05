import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, errorMessage, type Event } from '../api'
import { isLeader, useAuth } from '../auth'
import { formatDate } from '../format'

export default function EventsPage() {
  const { user } = useAuth()
  const leader = isLeader(user)
  const [past, setPast] = useState(false)
  const [events, setEvents] = useState<Event[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setEvents(null)
    api
      .get<Event[]>(`/events${past ? '?past=true' : ''}`)
      .then(setEvents)
      .catch((err) => setError(errorMessage(err)))
  }, [past])

  function replace(updated: Event) {
    setEvents((list) => list?.map((e) => (e.id === updated.id ? updated : e)) ?? null)
  }

  return (
    <div>
      <div className="page-header">
        <h1>{past ? 'Vergangene Events' : 'Kommende Events'}</h1>
        {leader && (
          <div className="actions">
            <button onClick={() => setPast(!past)}>{past ? 'Kommende' : 'Vergangene'}</button>
            <Link to="/events/neu" className="button primary">
              Neuer Event
            </Link>
          </div>
        )}
      </div>

      {error && <p className="error">{error}</p>}
      {events === null && !error && <p className="muted">Laden…</p>}
      {events?.length === 0 && <p className="muted">Keine Events vorhanden.</p>}

      <ul className="cards">
        {events?.map((e) => (
          <li key={e.id}>
            <EventCard event={e} leader={leader} onChange={replace} />
          </li>
        ))}
      </ul>
    </div>
  )
}

export function EventCard({
  event: e,
  leader,
  onChange,
}: {
  event: Event
  leader: boolean
  onChange: (e: Event) => void
}) {
  const [error, setError] = useState('')

  async function toggleRegistration() {
    setError('')
    try {
      const path = `/events/${e.id}/registration`
      onChange(e.registered ? await api.del<Event>(path) : await api.post<Event>(path))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <article className={`card event ${e.cancelled ? 'cancelled' : ''}`}>
      <Link to={`/events/${e.id}`} className="event-main">
        <div className="event-date">{formatDate(e.date)}</div>
        <div>
          {e.startTime} – {e.endTime} Uhr
        </div>
        <div className="muted">{e.location.name}</div>
      </Link>
      <div className="event-side">
        {e.cancelled && <span className="badge danger">Abgesagt</span>}
        {e.registered && !e.cancelled && <span className="badge success">Angemeldet</span>}
        {leader && (
          <span className="muted small">
            {e.registrationCount} angemeldet · {e.checkinCount} vor Ort
          </span>
        )}
        {e.registrationOpen && (
          <button className={e.registered ? '' : 'primary'} onClick={toggleRegistration}>
            {e.registered ? 'Abmelden' : 'Anmelden'}
          </button>
        )}
        {error && <span className="error small">{error}</span>}
      </div>
    </article>
  )
}
