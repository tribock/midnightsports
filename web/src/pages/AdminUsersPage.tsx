import { Fragment, useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { api, errorMessage, type EmergencyContact, type Role, type User, type UserSummary } from '../api'
import { isAdmin, useAuth } from '../auth'
import { PenIcon, PhoneIcon, TrashIcon } from '../components/Icons'
import ProfileFields, { type ProfileData } from '../components/ProfileFields'
import { relationLabels, roleLabels } from '../format'

/** Aufgeklappter Bereich unter einer Zeile: Notfallkontakt oder Bearbeiten */
type Panel = { userId: string; kind: 'emergency' | 'edit' }

/**
 * Benutzer: Admins vergeben Rollen, bearbeiten Profile (z.B. Email für Gäste) und löschen Benutzer.
 * Personen mit Zugriff auf Notfallkontakte sehen die Liste und Notfallkontakte.
 */
export default function AdminUsersPage() {
  const { user } = useAuth()
  const admin = isAdmin(user)
  const [query, setQuery] = useState('')
  const [users, setUsers] = useState<UserSummary[] | null>(null)
  const [panel, setPanel] = useState<Panel | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    // Antworten zu einem veralteten Suchbegriff ignorieren
    let current = true
    const t = setTimeout(() => {
      api
        .get<UserSummary[]>(`/users?q=${encodeURIComponent(query)}`)
        .then((list) => current && setUsers(list))
        .catch((err) => current && setError(errorMessage(err)))
    }, 250)
    return () => {
      current = false
      clearTimeout(t)
    }
  }, [query])

  function togglePanel(userId: string, kind: Panel['kind']) {
    setPanel((p) => (p?.userId === userId && p.kind === kind ? null : { userId, kind }))
  }

  function replace(updated: UserSummary) {
    setUsers((list) => list?.map((x) => (x.id === updated.id ? updated : x)) ?? null)
  }

  async function setRole(u: UserSummary, role: Role) {
    setError('')
    try {
      await api.put(`/users/${u.id}/role`, { role })
      replace({ ...u, role })
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove(u: UserSummary) {
    if (!confirm(`${displayName(u)} endgültig löschen? Besuche an Events bleiben anonym für die Statistik erhalten.`)) {
      return
    }
    setError('')
    try {
      await api.del(`/users/${u.id}`)
      setUsers((list) => list?.filter((x) => x.id !== u.id) ?? null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="stack">
      <div className="page-header">
        <h1>Benutzer</h1>
        {users && <span className="muted small">{users.length === 100 ? 'Erste 100 Treffer' : `${users.length} Benutzer`}</span>}
      </div>
      <input placeholder="Suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {error && <p className="error">{error}</p>}
      {users === null && !error && <p className="muted">Laden…</p>}
      {users?.length === 0 && <p className="muted">Keine Benutzer gefunden.</p>}

      {users && users.length > 0 && (
        <table className="table users">
          <thead>
            <tr>
              <th>Name</th>
              <th>Rolle</th>
              <th className="actions-cell">
                <span className="sr-only">Aktionen</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => {
              const open = panel?.userId === u.id ? panel.kind : null
              return (
                <Fragment key={u.id}>
                  <tr className={open ? 'open' : undefined}>
                    <td>
                      {displayName(u)}
                      {u.guest && <span className="badge">Gast</span>}
                      <br />
                      <span className="muted small">{u.email || 'keine Email'}</span>
                    </td>
                    <td>
                      {admin ? (
                        <select
                          aria-label={`Rolle von ${displayName(u)}`}
                          value={u.role}
                          disabled={u.id === user?.id}
                          onChange={(e) => setRole(u, e.target.value as Role)}
                        >
                          {Object.entries(roleLabels).map(([value, label]) => (
                            <option key={value} value={value}>
                              {label}
                            </option>
                          ))}
                        </select>
                      ) : (
                        roleLabels[u.role]
                      )}
                    </td>
                    <td className="actions-cell">
                      {user?.emergencyAccess && (
                        <IconButton
                          label={`Notfallkontakt von ${displayName(u)}`}
                          active={open === 'emergency'}
                          onClick={() => togglePanel(u.id, 'emergency')}
                        >
                          <PhoneIcon />
                        </IconButton>
                      )}
                      {admin && (
                        <IconButton
                          label={`${displayName(u)} bearbeiten`}
                          active={open === 'edit'}
                          onClick={() => togglePanel(u.id, 'edit')}
                        >
                          <PenIcon />
                        </IconButton>
                      )}
                      {admin && u.id !== user?.id && (
                        <IconButton label={`${displayName(u)} löschen`} danger onClick={() => remove(u)}>
                          <TrashIcon />
                        </IconButton>
                      )}
                    </td>
                  </tr>
                  {open && (
                    <tr className="panel-row">
                      <td colSpan={3}>
                        {open === 'emergency' ? (
                          <EmergencyContactPanel userId={u.id} />
                        ) : (
                          <EditUserPanel
                            userId={u.id}
                            onSaved={(saved) => {
                              replace({ ...u, firstName: saved.firstName, lastName: saved.lastName, email: saved.email })
                              setPanel(null)
                            }}
                            onCancel={() => setPanel(null)}
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

function displayName(u: UserSummary): string {
  return u.firstName || u.lastName ? `${u.firstName} ${u.lastName}`.trim() : '(Profil unvollständig)'
}

function IconButton({
  label,
  active,
  danger,
  onClick,
  children,
}: {
  label: string
  active?: boolean
  danger?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      className={['icon-button', active && 'active', danger && 'icon-danger'].filter(Boolean).join(' ')}
      aria-label={label}
      title={label}
      aria-expanded={active}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

/** Notfallkontakt wird erst beim Aufklappen geladen. */
function EmergencyContactPanel({ userId }: { userId: string }) {
  const [contact, setContact] = useState<EmergencyContact | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<EmergencyContact>(`/users/${userId}/emergency-contact`)
      .then(setContact)
      .catch((err) => setError(errorMessage(err)))
  }, [userId])

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

/** Profil eines Benutzers bearbeiten (nur Admins). Die Email ist nur bei Gästen änderbar. */
function EditUserPanel({
  userId,
  onSaved,
  onCancel,
}: {
  userId: string
  onSaved: (u: User) => void
  onCancel: () => void
}) {
  const [target, setTarget] = useState<User | null>(null)
  const [form, setForm] = useState<ProfileData | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<User>(`/users/${userId}`)
      .then((u) => {
        setTarget(u)
        setForm(u)
      })
      .catch((err) => setError(errorMessage(err)))
  }, [userId])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      onSaved(await api.put<User>(`/users/${userId}`, form))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (!target || !form) return error ? <p className="error small">{error}</p> : <p className="muted small">Laden…</p>
  return (
    <form className="stack" onSubmit={submit}>
      {target.guest && (
        <p className="muted small">
          Meldet sich der Gast mit der hier erfassten Email über Google an, übernimmt er dieses Konto.
        </p>
      )}
      <ProfileFields value={form} onChange={setForm} email={target.guest ? 'optional' : 'readonly'} canSetStaffStatus />
      {error && <p className="error">{error}</p>}
      <div className="actions">
        <button type="button" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="primary">
          Speichern
        </button>
      </div>
    </form>
  )
}
