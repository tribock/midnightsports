import { useEffect, useState } from 'react'
import { Link } from 'react-router'
import { errorMessage, type Func, type MemberDetails, type Role, type TeamMember } from '../api'
import { isAdmin } from '../auth'
import { formatDate, roleLabels, salaryLabel } from '../format'
import { useProject } from '../project'

/** Team: die Mitarbeiter (Status) des Projekts mit ihrer Funktion. Ändern nur durch Projekt-Admins. */
export default function TeamPage() {
  const { api, member } = useProject()
  const canEdit = isAdmin(member)
  const [team, setTeam] = useState<TeamMember[] | null>(null)
  const [functions, setFunctions] = useState<Func[]>([])
  const [error, setError] = useState('')
  // Person, deren Verlauf aufgeklappt ist
  const [open, setOpen] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([api.get<TeamMember[]>('/team'), api.get<Func[]>('/functions')])
      .then(([t, f]) => {
        setTeam(t)
        setFunctions(f)
      })
      .catch((err) => setError(errorMessage(err)))
  }, [api])

  async function setFunction(member: TeamMember, value: string) {
    const fn = functions.find((f) => String(f.id) === value)
    if (!fn) return
    setError('')
    try {
      await api.put(`/team/${member.id}/function`, { functionId: fn.id })
      setTeam((list) => list?.map((m) => (m.id === member.id ? { ...m, functionId: fn.id, functionName: fn.name } : m)) ?? null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function setRole(member: TeamMember, role: Role) {
    setError('')
    try {
      await api.put(`/members/${member.id}/role`, { role })
      setTeam((list) => list?.map((m) => (m.id === member.id ? { ...m, role } : m)) ?? null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  function label(f: Func) {
    return `${f.name} (${salaryLabel(f.salary)})`
  }

  return (
    <div className="stack">
      <h1>Team</h1>
      <p className="muted">
        Alle Mitarbeiter mit Rolle und Funktion an Events. Den Status Mitarbeiter vergeben Projekt-Admins unter
        Teilnehmer{canEdit ? ', Rolle und Funktion hier, Funktionen und Löhne unter Administration' : ''}.
      </p>
      {error && <p className="error">{error}</p>}
      {team === null && !error && <p className="muted">Laden…</p>}
      {team?.length === 0 && <p className="muted">Noch niemand im Team. Den Status Mitarbeiter unter Teilnehmer vergeben.</p>}
      {team && team.length > 0 && <FunctionOverview team={team} functions={functions} />}

      <ul className="list">
        {team?.map((m) => (
          <li key={m.id} className="team-member">
            <span className="grow">
              {m.name || m.email}
              {!canEdit && <span className="badge">{roleLabels[m.role]}</span>}
              <br />
              <span className="muted small">{m.email}</span>
            </span>
            {canEdit && (
              // Teilnehmer wird man nur, indem der Status Mitarbeiter entzogen wird (unter Teilnehmer)
              <select
                aria-label={`Rolle von ${m.name || m.email}`}
                value={m.role}
                disabled={m.id === member.id}
                onChange={(e) => setRole(m, e.target.value as Role)}
              >
                {(['mitarbeiter', 'projektleiter', 'admin'] as const).map((r) => (
                  <option key={r} value={r}>
                    {roleLabels[r]}
                  </option>
                ))}
              </select>
            )}
            {canEdit ? (
              <select
                aria-label={`Funktion von ${m.name || m.email}`}
                value={m.functionId}
                onChange={(e) => setFunction(m, e.target.value)}
              >
                {functions.map((f) => (
                  <option key={f.id} value={f.id}>
                    {label(f)}
                  </option>
                ))}
              </select>
            ) : (
              <span className="badge">{m.functionName}</span>
            )}
            <button
              type="button"
              className="small"
              aria-expanded={open === m.id}
              onClick={() => setOpen((id) => (id === m.id ? null : m.id))}
            >
              {open === m.id ? 'Verlauf schliessen' : 'Verlauf'}
            </button>
            {open === m.id && <MemberHistory userId={m.id} />}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Verlauf eines Mitarbeiters, erst beim Aufklappen geladen: Registrierung, Einsätze, Änderungen der Funktion */
function MemberHistory({ userId }: { userId: string }) {
  const { api, base } = useProject()
  const [details, setDetails] = useState<MemberDetails | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<MemberDetails>(`/members/${userId}/details`)
      .then(setDetails)
      .catch((err) => setError(errorMessage(err)))
  }, [api, userId])

  if (error) return <p className="error small member-history">{error}</p>
  if (!details) return <p className="muted small member-history">Laden…</p>
  return (
    <div className="stack member-history">
      <p className="muted small">
        Registriert am {dateOf(details.registered)}, im Projekt seit {dateOf(details.joined)} ·{' '}
        {details.eventsWorked} Einsätze geleistet·{' '}
        {details.eventsPlanned} Einsätze geplant· {details.eventsAttended}× als Teilnehmer/in eingecheckt
      </p>
      {details.events.length === 0 ? (
        <p className="muted small">Noch an keinem Event eingeteilt.</p>
      ) : (
        <ul className="list">
          {details.events.map((e) => (
            <li key={e.id}>
              <Link to={`${base}/events/${e.id}`} className="grow">
                {formatDate(e.date)}, {e.location.name}
              </Link>
              <span className="badge">{e.function}</span>
              {e.cancelled && <span className="badge danger">Abgesagt</span>}
            </li>
          ))}
        </ul>
      )}
      {details.functionChanges.length > 0 && (
        <><h2>Beförderungen</h2>
        <ul className="list small">
          {details.functionChanges.map((c, i) => (
            <li key={i}>
              <span className="grow">
                {dateOf(c.at)}: {c.from ?? 'kein Mitarbeiter'} → {c.to ?? 'kein Mitarbeiter'}
              </span>
              <span className="muted">{c.by}</span>
            </li>
          ))}
        </ul></>
      )}
    </div>
  )
}

function dateOf(iso: string) {
  return new Date(iso).toLocaleDateString('de-CH', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

/** Anzahl Personen pro Funktion, inkl. Funktionen ohne Personen */
function FunctionOverview({ team, functions }: { team: TeamMember[]; functions: Func[] }) {
  const count = (functionId: number) => team.filter((m) => m.functionId === functionId).length

  return (
    <dl className="stat-grid" aria-label="Personen pro Funktion">
      {functions.map((f) => (
        <div key={f.id} className="stat">
          <dt>{f.name}</dt>
          <dd>{count(f.id)}</dd>
        </div>
      ))}
      <div className="stat total">
        <dt>Total</dt>
        <dd>{team.length}</dd>
      </div>
    </dl>
  )
}
