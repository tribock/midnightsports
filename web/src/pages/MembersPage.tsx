import { Fragment, useCallback, useEffect, useState, type FormEvent, type ReactNode } from 'react'
import {
  errorMessage,
  type EmergencyContact,
  type MemberSummary,
  type Role,
  type SanctionLevel,
  type UserStatus,
} from '../api'
import { isAdmin, isProjectLead } from '../auth'
import { PenIcon, PhoneIcon } from '../components/Icons'
import { SanctionBadge, SanctionNotice } from '../components/Sanctions'
import StatusSelect from '../components/StatusSelect'
import { personName, relationLabels, roleLabels, statusLabels } from '../format'
import { useProject } from '../project'

/** Aufgeklappter Bereich unter einer Zeile: Notfallkontakt oder Verwarnung/Sperre */
type Panel = { userId: string; kind: 'emergency' | 'sanction' }

/**
 * Teilnehmer: die Mitglieder dieses Projekts. Projekt-Admins vergeben Rollen und Status, Projektleiter verwarnen
 * und sperren, Personen mit Zugriff sehen Notfallkontakte. Profile ändern nur globale Admins (Administration global).
 */
export default function MembersPage() {
  const { api, member, project } = useProject()
  const admin = isAdmin(member)
  const lead = isProjectLead(member)
  const [query, setQuery] = useState('')
  const [members, setMembers] = useState<MemberSummary[] | null>(null)
  const [panel, setPanel] = useState<Panel | null>(null)
  const [error, setError] = useState('')
  const [version, setVersion] = useState(0)
  const reload = useCallback(() => setVersion((v) => v + 1), [])

  useEffect(() => {
    // Antworten zu einem veralteten Suchbegriff ignorieren
    let current = true
    const t = setTimeout(() => {
      api
        .get<MemberSummary[]>(`/members?q=${encodeURIComponent(query)}`)
        .then((list) => current && setMembers(list))
        .catch((err) => current && setError(errorMessage(err)))
    }, 250)
    return () => {
      current = false
      clearTimeout(t)
    }
  }, [api, query, version])

  function togglePanel(userId: string, kind: Panel['kind']) {
    setPanel((p) => (p?.userId === userId && p.kind === kind ? null : { userId, kind }))
  }

  async function change(m: MemberSummary, field: 'role' | 'status', value: Role | UserStatus) {
    setError('')
    try {
      await api.put(`/members/${m.id}/${field}`, { [field]: value })
      setMembers((list) => list?.map((x) => (x.id === m.id ? { ...x, [field]: value } : x)) ?? null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  const labels = statusLabels(project.region)
  return (
    <div className="stack">
      <div className="page-header">
        <h1>Teilnehmer</h1>
        {members && (
          <span className="muted small">
            {members.length === 100
              ? 'Erste 100 Treffer'
              : members.length === 1
                ? '1 Mitglied'
                : `${members.length} Mitglieder`}
          </span>
        )}
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
              <th>Rolle / Status</th>
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
                      {m.guest && <span className="badge">Gast</span>}
                      <SanctionBadge sanctions={m.sanctions} />
                      <br />
                      <span className="muted small">{m.email || 'keine Email'}</span>
                    </td>
                    <td className="stack-cell">
                      {admin ? (
                        <>
                          <select
                            aria-label={`Rolle von ${personName(m)}`}
                            value={m.role}
                            disabled={m.id === member.id}
                            onChange={(e) => change(m, 'role', e.target.value as Role)}
                          >
                            {Object.entries(roleLabels).map(([value, label]) => (
                              <option key={value} value={value}>
                                {label}
                              </option>
                            ))}
                          </select>
                          <StatusSelect
                            value={m.status}
                            region={project.region}
                            canSetStaff
                            onChange={(status) => change(m, 'status', status)}
                          />
                        </>
                      ) : (
                        <>
                          {roleLabels[m.role]}
                          <br />
                          <span className="muted small">{m.status ? labels[m.status] : '–'}</span>
                        </>
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
                          <PenIcon />
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
                          <SanctionPanel
                            member={m}
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

/**
 * Verwarnen oder sperren (Projektleiter und Admins). Eine neue Verwarnung oder Sperre ersetzt die bisherige
 * dieses Projekts, die Frist beginnt neu. Geteilte Einträge anderer Projekte sind nur sichtbar.
 */
function SanctionPanel({ member: m, onSaved }: { member: MemberSummary; onSaved: () => void }) {
  const { api } = useProject()
  const own = m.sanctions.find((s) => s.own)
  const [level, setLevel] = useState<SanctionLevel>(own?.level ?? 'verwarnung')
  const [reason, setReason] = useState(own?.reason ?? '')
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.put(`/members/${m.id}/sanction`, { level, reason })
      onSaved()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove() {
    if (!confirm(`Verwarnung bzw. Sperre von ${personName(m)} aufheben?`)) return
    try {
      await api.del(`/members/${m.id}/sanction`)
      onSaved()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="stack" onSubmit={submit}>
      <SanctionNotice sanctions={m.sanctions} />
      <div className="row">
        <label>
          Massnahme
          <select value={level} onChange={(e) => setLevel(e.target.value as SanctionLevel)}>
            <option value="verwarnung">Verwarnen (Check-in mit Meldung)</option>
            <option value="sperre">Sperren (kein Check-in, keine Anmeldung)</option>
          </select>
        </label>
        <label className="grow">
          Grund
          <input
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="z.B. Regeln missachtet"
          />
        </label>
      </div>
      <p className="muted small">
        Wird beim Einchecken angezeigt. Fristen legt der Projekt-Admin unter Administration fest.
      </p>
      {error && <p className="error">{error}</p>}
      <div className="actions">
        {own && (
          <button type="button" className="danger" onClick={remove}>
            Aufheben
          </button>
        )}
        <span className="grow" />
        <button type="submit" className="primary">
          {own ? 'Ersetzen' : 'Speichern'}
        </button>
      </div>
    </form>
  )
}
