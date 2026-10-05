import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  api,
  errorMessage,
  type CheckinResult,
  type Event,
  type Participant,
  type Report,
  type UserSummary,
} from '../api'
import { isLeader, useAuth } from '../auth'
import { age, formatDate, formatTime } from '../format'
import { EventCard } from './EventsPage'

export default function EventDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const leader = isLeader(user)
  const [event, setEvent] = useState<Event | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<Event>(`/events/${id}`)
      .then(setEvent)
      .catch((err) => setError(errorMessage(err)))
  }, [id])

  async function setCancelled(cancelled: boolean) {
    if (cancelled && !confirm('Event absagen?')) return
    try {
      setEvent(await api.post<Event>(`/events/${id}/cancel`, { cancelled }))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (error) return <p className="error">{error}</p>
  if (!event) return <p className="muted">Laden…</p>

  const l = event.location
  return (
    <div className="stack">
      <Link to="/events" className="muted">
        ← Alle Events
      </Link>
      <h1>Event vom {formatDate(event.date)}</h1>
      <EventCard event={event} leader={leader} onChange={setEvent} />
      <p className="muted">
        {l.name}
        {l.street && `, ${l.street}`}
        {(l.zip || l.city) && `, ${l.zip} ${l.city}`}
        <br />
        Projektleitung: {event.leaders.map((p) => p.name).join(', ') || '–'}
      </p>

      {leader && (
        <>
          <div className="actions">
            {!event.cancelled && (
              <Link to={`/events/${event.id}/scan`} className="button primary">
                QR-Code scannen
              </Link>
            )}
            <Link to={`/events/${event.id}/bearbeiten`} className="button">
              Bearbeiten
            </Link>
            <button className={event.cancelled ? '' : 'danger'} onClick={() => setCancelled(!event.cancelled)}>
              {event.cancelled ? 'Wieder aktivieren' : 'Absagen'}
            </button>
          </div>
          <Participants eventId={event.id} disabled={event.cancelled} />
          <ReportEditor eventId={event.id} />
        </>
      )}
    </div>
  )
}

function Participants({ eventId, disabled }: { eventId: number; disabled: boolean }) {
  const [list, setList] = useState<Participant[]>([])
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<Participant[]>(`/events/${eventId}/participants`)
      .then(setList)
      .catch((err) => setError(errorMessage(err)))
  }, [eventId])

  useEffect(load, [load])

  async function undo(p: Participant) {
    if (!confirm(`Check-in von ${p.name} rückgängig machen?`)) return
    await api.del(`/events/${eventId}/checkins/${p.checkinId}`).catch((err) => setError(errorMessage(err)))
    load()
  }

  const present = list.filter((p) => p.checkedInAt).length
  return (
    <section className="stack">
      <h2>
        Teilnehmer <span className="muted small">({present} vor Ort)</span>
      </h2>
      {error && <p className="error">{error}</p>}
      {list.length === 0 && <p className="muted">Noch keine Anmeldungen.</p>}
      <ul className="list">
        {list.map((p) => (
          <li key={`${p.userId ?? ''}-${p.checkinId ?? ''}`}>
            <span className={p.checkedInAt ? 'dot success' : 'dot'} />
            <span className="grow">
              {p.name}
              {p.birthDate && <span className="muted"> · {age(p.birthDate)} J.</span>}
              {p.guest && <span className="badge">Gast</span>}
              {!p.registered && !p.guest && <span className="badge">nicht angemeldet</span>}
            </span>
            {p.checkedInAt ? (
              <button className="link small" onClick={() => undo(p)}>
                {formatTime(p.checkedInAt)} ✕
              </button>
            ) : (
              !disabled &&
              p.userId && <ManualCheckinButton eventId={eventId} userId={p.userId} onDone={load} />
            )}
          </li>
        ))}
      </ul>
      {!disabled && <ManualCheckin eventId={eventId} onDone={load} />}
    </section>
  )
}

function ManualCheckinButton({ eventId, userId, onDone }: { eventId: number; userId: string; onDone: () => void }) {
  return (
    <button className="small" onClick={() => api.post(`/events/${eventId}/checkins`, { userId }).then(onDone)}>
      Einchecken
    </button>
  )
}

/** Erfassung über die Eingabemaske: bestehenden Teilnehmer suchen oder Gast erfassen. */
function ManualCheckin({ eventId, onDone }: { eventId: number; onDone: () => void }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<UserSummary[]>([])
  const [guest, setGuest] = useState({ guestFirstName: '', guestLastName: '' })
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (query.trim().length < 2) {
      setResults([])
      return
    }
    const t = setTimeout(() => {
      api.get<UserSummary[]>(`/users?q=${encodeURIComponent(query)}`).then(setResults)
    }, 250)
    return () => clearTimeout(t)
  }, [query])

  async function checkIn(body: object) {
    setMessage('')
    try {
      const r = await api.post<CheckinResult>(`/events/${eventId}/checkins`, body)
      const name = r.person ? `${r.person.firstName} ${r.person.lastName}` : r.guestName
      setMessage(r.alreadyCheckedIn ? `${name} war bereits eingecheckt` : `${name} eingecheckt`)
      setQuery('')
      setGuest({ guestFirstName: '', guestLastName: '' })
      onDone()
    } catch (err) {
      setMessage(errorMessage(err))
    }
  }

  function submitGuest(e: FormEvent) {
    e.preventDefault()
    checkIn(guest)
  }

  return (
    <div className="card stack">
      <strong>Manuell erfassen</strong>
      <input placeholder="Teilnehmer suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {results.length > 0 && (
        <ul className="list">
          {results.map((u) => (
            <li key={u.id}>
              <span className="grow">
                {u.firstName} {u.lastName} <span className="muted small">{u.email}</span>
              </span>
              <button className="small" onClick={() => checkIn({ userId: u.id })}>
                Einchecken
              </button>
            </li>
          ))}
        </ul>
      )}
      <form className="row" onSubmit={submitGuest}>
        <input
          placeholder="Vorname Gast"
          required
          value={guest.guestFirstName}
          onChange={(e) => setGuest({ ...guest, guestFirstName: e.target.value })}
        />
        <input
          placeholder="Name Gast"
          required
          value={guest.guestLastName}
          onChange={(e) => setGuest({ ...guest, guestLastName: e.target.value })}
        />
        <button type="submit">Gast erfassen</button>
      </form>
      {message && <p className="small">{message}</p>}
    </div>
  )
}

function ReportEditor({ eventId }: { eventId: number }) {
  const [report, setReport] = useState<Report | null>(null)
  const [body, setBody] = useState('')
  const [status, setStatus] = useState('')

  useEffect(() => {
    api.get<Report>(`/events/${eventId}/report`).then((r) => {
      setReport(r)
      setBody(r.body)
    })
  }, [eventId])

  async function save(e: FormEvent) {
    e.preventDefault()
    try {
      const r = await api.put<Report>(`/events/${eventId}/report`, { body })
      setReport(r)
      setStatus('Gespeichert')
    } catch (err) {
      setStatus(errorMessage(err))
    }
  }

  if (!report) return null
  return (
    <section className="stack">
      <h2>Bericht</h2>
      <p className="muted small">
        {report.registrationCount} angemeldet · {report.checkinCount} vor Ort
        {report.updatedAt &&
          ` · zuletzt bearbeitet von ${report.authorName} am ${new Date(report.updatedAt).toLocaleString('de-CH')}`}
      </p>
      <form className="stack" onSubmit={save}>
        <textarea
          rows={8}
          value={body}
          placeholder="Ablauf, Vorkommnisse, Material, Bemerkungen…"
          onChange={(e) => {
            setBody(e.target.value)
            setStatus('')
          }}
        />
        <div className="actions">
          {status && <span className="muted small">{status}</span>}
          <button type="submit" className="primary">
            Bericht speichern
          </button>
        </div>
      </form>
    </section>
  )
}
