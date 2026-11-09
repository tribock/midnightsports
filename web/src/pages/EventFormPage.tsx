import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  errorMessage,
  type Event,
  type EventInput,
  type Func,
  type Location,
  type StaffEntry,
  type TeamMember,
} from '../api'
import { isProjectLead } from '../auth'
import { formatCHF, roleLabels, todayISO } from '../format'
import { useProject } from '../project'

export default function EventFormPage() {
  const { id } = useParams()
  const { api, base, member: user } = useProject()
  const navigate = useNavigate()
  const canStaff = isProjectLead(user)
  const [locations, setLocations] = useState<Location[]>([])
  const [team, setTeam] = useState<TeamMember[]>([])
  const [functions, setFunctions] = useState<Func[]>([])
  const [form, setForm] = useState({
    date: todayISO(),
    startTime: '21:00',
    endTime: '00:00',
    locationId: 0,
    leaderId: '',
  })
  // Einsatz: ausgewählte Personen und bisher gespeicherte Einträge (für die Anzeige des Betrags)
  const [staff, setStaff] = useState<Set<string>>(new Set())
  const [existing, setExisting] = useState<StaffEntry[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([
      api.get<Location[]>('/locations'),
      api.get<TeamMember[]>('/team'),
      canStaff ? api.get<Func[]>('/functions') : Promise.resolve([]),
      id ? api.get<Event>(`/events/${id}`) : Promise.resolve(null),
      id && canStaff ? api.get<StaffEntry[]>(`/events/${id}/staff`) : Promise.resolve([]),
    ])
      .then(([locs, members, fns, event, existing]) => {
        setLocations(locs)
        setTeam(members)
        setFunctions(fns)
        const leaders = members.filter(isProjectLead)
        if (event) {
          setForm({
            date: event.date,
            startTime: event.startTime,
            endTime: event.endTime,
            locationId: event.location.id,
            leaderId: event.leader?.id ?? '',
          })
          setStaff(new Set(existing.map((s) => s.userId)))
          setExisting(existing)
        } else {
          setForm((f) => ({
            ...f,
            locationId: locs.find((l) => l.isDefault)?.id ?? locs[0]?.id ?? 0,
            leaderId: leaders.some((m) => m.id === user!.id) ? user!.id : '',
          }))
        }
      })
      .catch((err) => setError(errorMessage(err)))
  }, [api, id, canStaff, user])

  // Verantwortlich sein können Projektleiter und Admins
  const leaders = team.filter(isProjectLead)

  /** Funktion ohne festen Lohn: Abrechnung über die Zeiterfassung, kein Betrag am Event */
  function paidByTime(m: TeamMember): boolean {
    return functions.some((f) => f.id === m.functionId && f.salary === null)
  }

  /** Betrag am Event: bisher gespeicherter Betrag bei gleicher Funktion, sonst Lohn der Funktion */
  function salary(m: TeamMember): number | null {
    const prev = existing.find((s) => s.userId === m.id)
    if (prev && prev.functionId === m.functionId && prev.salary != null && !paidByTime(m)) return prev.salary
    return functions.find((f) => f.id === m.functionId)?.salary ?? null
  }

  function toggleStaff(m: TeamMember) {
    setStaff((s) => {
      const next = new Set(s)
      if (next.has(m.id)) next.delete(m.id)
      else next.add(m.id)
      return next
    })
  }

  // Der verantwortliche Projektleiter gehört immer zum Team, sofern er eine Funktion hat.
  // Wechselt die Projektleitung, gilt das für die neue Person, die bisherige bleibt nur, wenn sie angekreuzt wurde.
  const responsible = team.find((m) => m.id === form.leaderId)
  const isResponsible = (m: TeamMember) => m.id === responsible?.id && !!m.functionId
  const selectedTeam = team.filter((m) => staff.has(m.id) || isResponsible(m))

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    const body: EventInput = { ...form }
    if (canStaff) {
      // Personen, die nicht mehr im Team sind, fallen aus dem Einsatz
      body.staff = selectedTeam.map((m) => ({ userId: m.id }))
    }
    try {
      const saved = id ? await api.put<Event>(`/events/${id}`, body) : await api.post<Event>('/events', body)
      navigate(`${base}/events/${saved.id}`)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  const total = selectedTeam.reduce((sum, m) => sum + (salary(m) ?? 0), 0)
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
        <label>
          Projektleiter (verantwortlich)
          <select required value={form.leaderId} onChange={(e) => setForm({ ...form, leaderId: e.target.value })}>
            <option value="">Bitte wählen</option>
            {leaders.map((m) => (
              <option key={m.id} value={m.id}>
                {memberLabel(m)}
              </option>
            ))}
          </select>
        </label>

        {canStaff && (
          <fieldset className="stack">
            <legend>Team</legend>
            {team.length === 0 && <p className="muted small">Noch niemand im Team.</p>}
            {team.map((m) => {
              const selected = selectedTeam.includes(m)
              const locked = isResponsible(m)
              return (
                <div key={m.id} className="staff-row">
                  <label className="check grow">
                    <input
                      type="checkbox"
                      checked={selected}
                      disabled={locked || (!m.functionId && !selected)}
                      onChange={() => toggleStaff(m)}
                    />
                    <span>
                      {m.name || m.email}
                      {m.functionName ? (
                        <span className="badge">{m.functionName}</span>
                      ) : (
                        <span className="muted small"> · keine Funktion</span>
                      )}
                      {locked && <span className="muted small"> · verantwortlich</span>}
                    </span>
                  </label>
                  {selected && (
                    <span className="muted small salary">
                      {paidByTime(m) ? 'über Zeiterfassung' : formatCHF(salary(m) ?? 0)}
                    </span>
                  )}
                </div>
              )
            })}
            {responsible && !responsible.functionId && (
              <p className="muted small">
                {responsible.name || responsible.email} hat keine Funktion und kann nicht eingeteilt werden (Team).
              </p>
            )}
            {selectedTeam.length > 0 && (
              <p className="staff-total">
                <span>Total</span>
                <strong>{formatCHF(total)}</strong>
              </p>
            )}
            <p className="muted small">
              Funktion und Betrag kommen aus Team und Administration. Den Betrag sehen nur Projektleiter und Admins.
            </p>
          </fieldset>
        )}

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

/** "Louis Kern · Admin · PL", damit gleichnamige Personen unterscheidbar sind */
function memberLabel(m: TeamMember): string {
  return [m.name || m.email, roleLabels[m.role], m.functionName].filter(Boolean).join(' · ')
}
