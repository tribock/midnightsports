import { useEffect, useState } from 'react'
import { errorMessage, type Func, type TeamMember } from '../api'
import { isAdmin } from '../auth'
import { roleLabels, salaryLabel } from '../format'
import { useProject } from '../project'

/** Team: die Mitarbeiter (Status) des Projekts mit ihrer Funktion. Ändern nur durch Projekt-Admins. */
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
    if (!fn) return
    setError('')
    try {
      await api.put(`/team/${member.id}/function`, { functionId: fn.id })
      setTeam((list) => list?.map((m) => (m.id === member.id ? { ...m, functionId: fn.id, functionName: fn.name } : m)) ?? null)
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
        Alle Mitarbeiter mit ihrer Funktion an Events. Den Status Mitarbeiter samt Funktion vergeben Projekt-Admins
        unter Teilnehmer{canEdit ? ', Funktionen und Löhne unter Administration' : ''}.
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
              <span className="badge">{roleLabels[m.role]}</span>
              <br />
              <span className="muted small">{m.email}</span>
            </span>
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
          </li>
        ))}
      </ul>
    </div>
  )
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
