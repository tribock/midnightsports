import { Fragment, useCallback, useEffect, useState, type ReactNode } from 'react'
import {
  errorMessage,
  type EmergencyContact,
  type Func,
  type MemberSummary,
  type Page,
  type Role,
  type UserStatus,
} from '../api'
import { isAdmin, isProjectLead } from '../auth'
import { PhoneIcon, WarningIcon } from '../components/Icons'
import { SanctionBadge, SanctionForm } from '../components/Sanctions'
import StatusSelect from '../components/StatusSelect'
import { personName, relationLabels, roleLabels, statusLabels } from '../format'
import { Pager, usePagedSearch } from '../paging'
import { useProject } from '../project'

/** Aufgeklappter Bereich unter einer Zeile: Notfallkontakt oder Verwarnung/Sperre */
type Panel = { userId: string; kind: 'emergency' | 'sanction' }

/**
 * Teilnehmer: die Mitglieder dieses Projekts. Projekt-Admins vergeben den Status (Mitarbeiter samt Funktion,
 * Rollen danach im Team), Projektleiter verwarnen
 * und sperren, Personen mit Zugriff sehen Notfallkontakte. Profile ändern nur globale Admins (Administration global).
 */
export default function MembersPage() {
  const { api, member, project } = useProject()
  const admin = isAdmin(member)
  const lead = isProjectLead(member)
  const [panel, setPanel] = useState<Panel | null>(null)
  // Funktionen für neue Mitarbeiter; staffing: Person, für die gerade die Funktion gewählt wird
  const [functions, setFunctions] = useState<Func[]>([])
  const [staffing, setStaffing] = useState<string | null>(null)
  const fetchPage = useCallback(
    (q: string, offset: number) => api.get<Page<MemberSummary>>(`/members?q=${encodeURIComponent(q)}&offset=${offset}`),
    [api],
  )
  const { query, setQuery, page, setOffset, error, setError, updateItems, reload } = usePagedSearch(fetchPage)
  const members = page?.items ?? null

  function togglePanel(userId: string, kind: Panel['kind']) {
    setPanel((p) => (p?.userId === userId && p.kind === kind ? null : { userId, kind }))
  }

  useEffect(() => {
    if (!admin) return
    api
      .get<Func[]>('/functions')
      .then(setFunctions)
      .catch((err) => setError(errorMessage(err)))
  }, [api, admin, setError])

  // Der Status bestimmt die Rolle: Mitarbeiter beginnen als Mitarbeiter (Rolle und Funktion ändern sich danach im
  // Team), alle anderen sind Teilnehmer
  async function saveStatus(m: MemberSummary, status: UserStatus, functionId?: number) {
    setError('')
    try {
      await api.put(`/members/${m.id}/status`, { status, functionId })
      const role: Role = status === 'mitarbeiter' ? 'mitarbeiter' : 'teilnehmer'
      updateItems((list) => list.map((x) => (x.id === m.id ? { ...x, status, role } : x)))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  // Mitarbeiter brauchen eine Funktion: erst nach deren Wahl wird gespeichert. Wer den Status verliert, verliert
  // auch Rolle und Funktion und damit den Platz im Team.
  function changeStatus(m: MemberSummary, status: UserStatus) {
    setStaffing(status === 'mitarbeiter' && m.status !== 'mitarbeiter' ? m.id : null)
    if (status === 'mitarbeiter' || status === m.status) return
    if (
      m.status === 'mitarbeiter' &&
      !confirm(`${personName(m)} verliert Rolle und Funktion und ist nicht mehr im Team. Status ändern?`)
    )
      return
    saveStatus(m, status)
  }

  function chooseFunction(m: MemberSummary, value: string) {
    setStaffing(null)
    if (value) saveStatus(m, 'mitarbeiter', Number(value))
  }

  const labels = statusLabels(project.region)
  return (
    <div className="stack">
      <div className="page-header">
        <h1>Teilnehmer</h1>
        {page && <span className="muted small">{page.total === 1 ? '1 Mitglied' : `${page.total} Mitglieder`}</span>}
      </div>
      <p className="muted small">
        Alle Personen, die sich für {project.name} angemeldet haben. Profile ändert nur die globale Administration.
      </p>
      <input placeholder="Suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {error && <p className="error">{error}</p>}
      {members === null && !error && <p className="muted">Laden…</p>}
      {members?.length === 0 && <p className="muted">Keine Mitglieder gefunden.</p>}

      {members && members.length > 0 && (
        <table className="table users">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th className="actions-cell">
                <span className="sr-only">Aktionen</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => {
              const open = panel?.userId === m.id ? panel.kind : null
              return (
                <Fragment key={m.id}>
                  <tr className={open ? 'open' : undefined}>
                    <td>
                      {personName(m)}
                      {m.role !== 'teilnehmer' && <span className="badge">{roleLabels[m.role]}</span>}
                      {m.guest && <span className="badge">Gast</span>}
                      <SanctionBadge sanctions={m.sanctions} />
                      <br />
                      <span className="muted small">{m.email || 'keine Email'}</span>
                    </td>
                    <td>
                      {admin ? (
                        <div className="stack-cell">
                          <StatusSelect
                            value={staffing === m.id ? 'mitarbeiter' : m.status}
                            region={project.region}
                            canSetStaff
                            disabled={m.id === member.id}
                            onChange={(status) => changeStatus(m, status)}
                          />
                          {staffing === m.id && (
                            <label>
                              Funktion
                              <select
                                required
                                aria-label={`Funktion von ${personName(m)}`}
                                value=""
                                onChange={(e) => chooseFunction(m, e.target.value)}
                              >
                                <option value="" disabled>
                                  Bitte wählen
                                </option>
                                {functions.map((f) => (
                                  <option key={f.id} value={f.id}>
                                    {f.name}
                                  </option>
                                ))}
                              </select>
                            </label>
                          )}
                        </div>
                      ) : (
                        <span className="muted small">{m.status ? labels[m.status] : '–'}</span>
                      )}
                    </td>
                    <td className="actions-cell">
                      {member.emergencyAccess && (
                        <IconButton
                          label={`Notfallkontakt von ${personName(m)}`}
                          active={open === 'emergency'}
                          onClick={() => togglePanel(m.id, 'emergency')}
                        >
                          <PhoneIcon />
                        </IconButton>
                      )}
                      {lead && m.id !== member.id && (
                        <IconButton
                          label={`${personName(m)} verwarnen oder sperren`}
                          active={open === 'sanction'}
                          onClick={() => togglePanel(m.id, 'sanction')}
                        >
                          <WarningIcon />
                        </IconButton>
                      )}
                    </td>
                  </tr>
                  {open && (
                    <tr className="panel-row">
                      <td colSpan={3}>
                        {open === 'emergency' ? (
                          <EmergencyContactPanel userId={m.id} />
                        ) : (
                          <SanctionForm
                            userId={m.id}
                            name={personName(m)}
                            sanctions={m.sanctions}
                            onSaved={() => {
                              setPanel(null)
                              reload()
                            }}
                          />
                        )}
                      </td>
                    </tr>
                  )}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      )}
      {page && page.total > 0 && <Pager page={page} onChange={setOffset} />}
    </div>
  )
}

function IconButton({
  label,
  active,
  onClick,
  children,
}: {
  label: string
  active?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      className={['icon-button', active && 'active'].filter(Boolean).join(' ')}
      aria-label={label}
      title={label}
      aria-expanded={active}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

/** Notfallkontakt wird erst beim Aufklappen geladen. Jeder Abruf wird im Audit-Protokoll festgehalten. */
function EmergencyContactPanel({ userId }: { userId: string }) {
  const { api } = useProject()
  const [contact, setContact] = useState<EmergencyContact | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<EmergencyContact>(`/members/${userId}/emergency-contact`)
      .then(setContact)
      .catch((err) => setError(errorMessage(err)))
  }, [api, userId])

  if (error) return <p className="error small">{error}</p>
  if (!contact) return <p className="muted small">Laden…</p>
  if (!contact.phone) return <p className="muted small">Kein Notfallkontakt erfasst</p>
  return (
    <p className="small">
      <strong>Notfallkontakt:</strong> {contact.name}
      {contact.relation && <span className="muted"> ({relationLabels[contact.relation]})</span>} ·{' '}
      <a href={`tel:${contact.phone}`}>{contact.phone}</a>
    </p>
  )
}
