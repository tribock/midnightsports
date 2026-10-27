import { useEffect, useState } from 'react'
import { errorMessage, type Func, type TeamMember } from '../api'
import { isAdmin } from '../auth'
import { roleLabels, salaryLabel } from '../format'
import { useProject } from '../project'

/** Team: Projektleiter, Mitarbeiter und Admins des Projekts mit ihrer Funktion. Ändern nur durch Projekt-Admins. */
export default function TeamPage() {
  const { api, member } = useProject()
  const canEdit = isAdmin(member)
  const [team, setTeam] = useState<TeamMember[] | null>(null)
  const [functions, setFunctions] = useState<Func[]>([])
  const [error, setError] = useState('')

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
    setError('')
    try {
      await api.put(`/team/${member.id}/function`, { functionId: fn ? fn.id : null })
      setTeam(
        (list) =>
          list?.map((m) => (m.id === member.id ? { ...m, functionId: fn?.id ?? null, functionName: fn?.name ?? null } : m)) ??
          null,
      )
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
        Projektleiter, Mitarbeiter und Admins mit ihrer Funktion an Events. Rollen werden unter Teilnehmer vergeben
        {canEdit ? ', Funktionen und Löhne unter Administration' : ''}.
      </p>
      {error && <p className="error">{error}</p>}
      {team === null && !error && <p className="muted">Laden…</p>}
      {team?.length === 0 && <p className="muted">Noch niemand im Team. Rollen unter Teilnehmer vergeben.</p>}
      {team && team.length > 0 && <FunctionOverview team={team} functions={functions} />}

      <ul className="list">
        {team?.map((m) => (
          <li key={m.id} className="team-member">
            <span className="grow">
              {m.name || m.email}
              <span className="badge">{roleLabels[m.role]}</span>
              <br />
              <span className="muted small">{m.email}</span>
            </span>
            {canEdit ? (
              <select
                aria-label={`Funktion von ${m.name || m.email}`}
                value={m.functionId ?? ''}
                onChange={(e) => setFunction(m, e.target.value)}
              >
                <option value="">Keine Funktion</option>
                {functions.map((f) => (
                  <option key={f.id} value={f.id}>
                    {label(f)}
                  </option>
                ))}
              </select>
            ) : (
              <span className={m.functionName ? 'badge' : 'muted small'}>{m.functionName ?? 'Keine Funktion'}</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Anzahl Personen pro Funktion, inkl. Funktionen ohne Personen und Personen ohne Funktion */
function FunctionOverview({ team, functions }: { team: TeamMember[]; functions: Func[] }) {
  const count = (functionId: number | null) => team.filter((m) => m.functionId === functionId).length
  const withoutFunction = count(null)

  return (
    <dl className="stat-grid" aria-label="Personen pro Funktion">
      {functions.map((f) => (
        <div key={f.id} className="stat">
          <dt>{f.name}</dt>
          <dd>{count(f.id)}</dd>
        </div>
      ))}
      {withoutFunction > 0 && (
        <div className="stat muted">
          <dt>Ohne Funktion</dt>
          <dd>{withoutFunction}</dd>
        </div>
      )}
      <div className="stat total">
        <dt>Total</dt>
        <dd>{team.length}</dd>
      </div>
    </dl>
  )
}
