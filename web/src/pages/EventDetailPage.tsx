import { Fragment, useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  errorMessage,
  type CheckinResult,
  type Event,
  type MemberSummary,
  type Participant,
  type Report,
  type StaffEntry,
  type UserStatus,
} from '../api'
import { isAdmin, isLeader, isProjectLead } from '../auth'
import { age, formatAddress, formatCHF, formatDate, formatTime, personName, statusLabels } from '../format'
import EventInfo from '../components/EventInfo'
import PrintSheet, { printDocument } from '../components/PrintSheet'
import ProfileFields, { emptyProfile, type ProfileData } from '../components/ProfileFields'
import { WarningIcon } from '../components/Icons'
import { SanctionBadge, SanctionForm, SanctionNotice } from '../components/Sanctions'
import StatusSelect from '../components/StatusSelect'
import { useProject } from '../project'
import { EventCard } from './EventsPage'

export default function EventDetailPage() {
  const { id } = useParams()
  const { api, base, member } = useProject()
  const leader = isLeader(member)
  const [event, setEvent] = useState<Event | null>(null)
  // Einsatz wird einmal geladen und von Einsatz und Bericht gemeinsam verwendet
  const [staff, setStaff] = useState<StaffEntry[]>([])
  const [staffError, setStaffError] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<Event>(`/events/${id}`)
      .then(setEvent)
      .catch((err) => setError(errorMessage(err)))
  }, [api, id])

  useEffect(() => {
    if (!leader) return
    api
      .get<StaffEntry[]>(`/events/${id}/staff`)
      .then(setStaff)
      .catch((err) => setStaffError(errorMessage(err)))
  }, [api, id, leader])

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

  return (
    <div className="stack">
      <Link to={`${base}/events`} className="muted">
        ← Alle Events
      </Link>
      <h1>Event vom {formatDate(event.date)}</h1>
      <EventCard event={event} leader={leader} onChange={setEvent} />
      <EventInfo event={event} />

      {leader && (
        <>
          <div className="actions">
            {!event.cancelled && (
              <Link to={`${base}/events/${event.id}/scan`} className="button primary">
                QR-Code scannen
              </Link>
            )}
            {isProjectLead(member) && (
              <>
                <Link to={`${base}/events/${event.id}/bearbeiten`} className="button">
                  Bearbeiten
                </Link>
                <Link to={`${base}/events/${event.id}/auszahlung`} className="button">
                  Auszahlung
                </Link>
                <button className={event.cancelled ? '' : 'danger'} onClick={() => setCancelled(!event.cancelled)}>
                  {event.cancelled ? 'Wieder aktivieren' : 'Absagen'}
                </button>
              </>
            )}
          </div>
          <Staff staff={staff} error={staffError} />
          <Participants eventId={event.id} disabled={event.cancelled} />
          {isProjectLead(member) && <ReportEditor event={event} staff={staff} />}
        </>
      )}
    </div>
  )
}

/** Einsatz: Team-Mitglieder mit Funktion und Lohn (Lohn nur für Projektleiter und Admins). Bearbeiten im Event-Formular. */
function Staff({ staff, error }: { staff: StaffEntry[]; error: string }) {
  const withSalary = staff.some((s) => s.salary !== undefined)
  const total = staff.reduce((sum, s) => sum + (s.salary ?? 0), 0)
  return (
    <section className="stack">
      <h2>Einsatz</h2>
      {error && <p className="error">{error}</p>}
      {staff.length === 0 && <p className="muted">Noch niemand eingeteilt (Event bearbeiten).</p>}
      {staff.length > 0 && (
        <ul className="list">
          {staff.map((s) => (
            <li key={s.userId}>
              <span className="grow">
                {s.name}
                <span className="badge">{s.functionName}</span>
              </span>
              {s.salary !== undefined && <span className="num">{formatCHF(s.salary)}</span>}
              {s.paidByTime && <span className="muted small">über Zeiterfassung</span>}
            </li>
          ))}
          {withSalary && (
            <li>
              <strong className="grow">Total</strong>
              <strong className="num">{formatCHF(total)}</strong>
            </li>
          )}
        </ul>
      )}
    </section>
  )
}

function Participants({ eventId, disabled }: { eventId: number; disabled: boolean }) {
  const { api, project, member } = useProject()
  const canSanction = isProjectLead(member)
  const [list, setList] = useState<Participant[]>([])
  // Teilnehmer, dessen Formular zum Verwarnen offen ist
  const [sanctioning, setSanctioning] = useState<string | null>(null)
  const [error, setError] = useState('')
  // Ergebnis des letzten Check-ins aus der Liste, mit Meldung bei Verwarnung oder Sperre
  const [last, setLast] = useState<CheckinResult | null>(null)

  const load = useCallback(() => {
    api
      .get<Participant[]>(`/events/${eventId}/participants`)
      .then(setList)
      .catch((err) => setError(errorMessage(err)))
  }, [api, eventId])

  useEffect(load, [load])

  async function checkIn(userId: string) {
    setError('')
    try {
      setLast(await api.post<CheckinResult>(`/events/${eventId}/checkins`, { userId }))
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

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
      {last && <CheckinMessage result={last} />}
      {list.length === 0 && <p className="muted">Noch keine Anmeldungen.</p>}
      <ul className="list">
        {list.map((p) => (
          <Fragment key={`${p.userId ?? ''}-${p.checkinId ?? ''}`}>
            <li>
              <span className={p.checkedInAt ? 'dot success' : 'dot'} />
              <span className="grow">
                {p.name}
                {p.birthDate && <span className="muted"> · {age(p.birthDate)} J.</span>}
                {p.status && <span className="muted small"> · {statusLabels(project.region)[p.status]}</span>}
                {p.guest && <span className="badge">Gast</span>}
                <SanctionBadge sanctions={p.sanctions} />
                {!p.registered && !p.guest && <span className="badge">nicht angemeldet</span>}
              </span>
              {p.checkedInAt ? (
                <button className="link small" onClick={() => undo(p)}>
                  {formatTime(p.checkedInAt)} ✕
                </button>
              ) : (
                !disabled &&
                p.userId && (
                  <button className="small" onClick={() => checkIn(p.userId!)}>
                    Einchecken
                  </button>
                )
              )}
              {canSanction && p.userId && (
                <button
                  className={sanctioning === p.userId ? 'icon-button active' : 'icon-button'}
                  aria-label={`${p.name} verwarnen oder sperren`}
                  title={`${p.name} verwarnen oder sperren`}
                  aria-expanded={sanctioning === p.userId}
                  onClick={() => setSanctioning(sanctioning === p.userId ? null : p.userId)}
                >
                  <WarningIcon />
                </button>
              )}
            </li>
            {sanctioning && sanctioning === p.userId && (
              <li className="editing">
                <SanctionForm
                  userId={p.userId}
                  name={p.name}
                  sanctions={p.sanctions}
                  onSaved={() => {
                    setSanctioning(null)
                    load()
                  }}
                />
              </li>
            )}
          </Fragment>
        ))}
      </ul>
      {!disabled && <ManualCheckin eventId={eventId} onDone={load} />}
    </section>
  )
}

/** Ergebnis eines Check-ins mit Meldung bei Verwarnung (eingecheckt) oder Sperre (nicht eingecheckt). */
function CheckinMessage({ result }: { result: CheckinResult }) {
  const name = personName(result.person)
  const text = result.blocked
    ? `${name} ist gesperrt und wurde nicht eingecheckt`
    : result.alreadyCheckedIn
      ? `${name} war bereits eingecheckt`
      : `${name} eingecheckt`
  return (
    <div className="stack">
      <p className="small">{text}</p>
      <SanctionNotice sanctions={result.sanctions} />
    </div>
  )
}

/** Erfassung über die Eingabemaske: Mitglied des Projekts suchen oder Gast erfassen. */
function ManualCheckin({ eventId, onDone }: { eventId: number; onDone: () => void }) {
  const { api } = useProject()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<MemberSummary[]>([])
  const [guestOpen, setGuestOpen] = useState(false)
  const [result, setResult] = useState<CheckinResult | null>(null)
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (query.trim().length < 2) {
      setResults([])
      return
    }
    // Antworten zu einem veralteten Suchbegriff ignorieren
    let current = true
    const t = setTimeout(() => {
      api.get<MemberSummary[]>(`/members?q=${encodeURIComponent(query)}`).then((list) => current && setResults(list))
    }, 250)
    return () => {
      current = false
      clearTimeout(t)
    }
  }, [api, query])

  function done(r: CheckinResult) {
    setResult(r)
    setMessage('')
    setQuery('')
    setGuestOpen(false)
    onDone()
  }

  async function checkIn(userId: string) {
    setMessage('')
    setResult(null)
    try {
      done(await api.post<CheckinResult>(`/events/${eventId}/checkins`, { userId }))
    } catch (err) {
      setMessage(errorMessage(err))
    }
  }

  return (
    <div className="card stack">
      <strong>Manuell erfassen</strong>
      <input placeholder="Mitglied suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {results.length > 0 && (
        <ul className="list">
          {results.map((u) => (
            <li key={u.id}>
              <span className="grow">
                {u.firstName} {u.lastName}
                {u.guest && <span className="badge">Gast</span>}
                <SanctionBadge sanctions={u.sanctions} /> <span className="muted small">{u.email}</span>
              </span>
              <button className="small" onClick={() => checkIn(u.id)}>
                Einchecken
              </button>
            </li>
          ))}
        </ul>
      )}
      {message && <p className="small">{message}</p>}
      {result && <CheckinMessage result={result} />}
      {guestOpen ? (
        <GuestForm eventId={eventId} onDone={done} onCancel={() => setGuestOpen(false)} />
      ) : (
        <div className="actions">
          <button onClick={() => setGuestOpen(true)}>Neuen Gast erfassen</button>
        </div>
      )}
    </div>
  )
}

/** Gast ohne Konto: dieselben Angaben wie im Profil, Email freiwillig, plus Status im Projekt.
 * Wird als Person gespeichert und beim Projekt angemeldet. */
function GuestForm({
  eventId,
  onDone,
  onCancel,
}: {
  eventId: number
  onDone: (r: CheckinResult) => void
  onCancel: () => void
}) {
  const { api, member, project } = useProject()
  const [form, setForm] = useState<ProfileData>(emptyProfile)
  const [status, setStatus] = useState<UserStatus | ''>('')
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      onDone(await api.post<CheckinResult>(`/events/${eventId}/guests`, { ...form, status }))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="stack" onSubmit={submit}>
      <strong>Neuer Gast</strong>
      <p className="muted small">
        Der Gast wird als Mitglied des Projekts gespeichert und kann an späteren Events über die Suche eingecheckt
        werden.
      </p>
      <ProfileFields value={form} onChange={setForm} email="optional" />
      <StatusSelect value={status} onChange={setStatus} region={project.region} canSetStaff={isAdmin(member)} />
      {error && <p className="error">{error}</p>}
      <div className="actions">
        <button type="button" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="primary">
          Gast erfassen und einchecken
        </button>
      </div>
    </form>
  )
}

function ReportEditor({ event, staff }: { event: Event; staff: StaffEntry[] }) {
  const { api } = useProject()
  const [report, setReport] = useState<Report | null>(null)
  const [body, setBody] = useState('')
  const [status, setStatus] = useState('')

  useEffect(() => {
    api.get<Report>(`/events/${event.id}/report`).then((r) => {
      setReport(r)
      setBody(r.body)
    })
  }, [api, event.id])

  async function save(e: FormEvent) {
    e.preventDefault()
    try {
      const r = await api.put<Report>(`/events/${event.id}/report`, { body })
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
          <button type="button" onClick={() => printDocument(`Bericht_${event.date}`)}>
            Bericht drucken
          </button>
          <button type="submit" className="primary">
            Bericht speichern
          </button>
        </div>
      </form>
      <ReportSheet event={event} report={report} staff={staff} body={body} />
    </section>
  )
}

/** Bericht zum Drucken bzw. als PDF. Gedruckt wird der aktuelle Text im Eingabefeld. */
function ReportSheet({
  event,
  report,
  staff,
  body,
}: {
  event: Event
  report: Report
  staff: StaffEntry[]
  body: string
}) {
  const a = report.attendance
  const address = formatAddress(event.location)
  return (
    <PrintSheet title="Bericht">
      <dl className="sheet-fields">
        <dt>Datum</dt>
        <dd>
          {formatDate(event.date)}, {event.startTime}–{event.endTime} Uhr
        </dd>
        <dt>Ort</dt>
        <dd>
          {event.location.name}
          {address && `, ${address}`}
        </dd>
        <dt>Zuständiger Projektleiter</dt>
        <dd>{event.leader?.name || '–'}</dd>
        <dt>Teilnehmer</dt>
        <dd>
          {report.checkinCount} vor Ort (W {a.w} · M {a.m} · D {a.d}
          {a.other > 0 && ` · ohne Angabe ${a.other}`}), {report.registrationCount} angemeldet
        </dd>
      </dl>

      <h2 className="sheet-heading">Anwesendes Team</h2>
      {staff.length === 0 ? (
        <p>Niemand eingeteilt.</p>
      ) : (
        <table className="sheet-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Funktion</th>
            </tr>
          </thead>
          <tbody>
            {staff.map((s) => (
              <tr key={s.userId}>
                <td>{s.name}</td>
                <td>{s.functionName}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h2 className="sheet-heading">Bericht</h2>
      <p className="sheet-report">{body || '–'}</p>

      <p className="sheet-small">
        {report.updatedAt
          ? `Zuletzt bearbeitet von ${report.authorName} am ${new Date(report.updatedAt).toLocaleString('de-CH')}`
          : 'Noch nicht gespeichert'}
      </p>
    </PrintSheet>
  )
}
