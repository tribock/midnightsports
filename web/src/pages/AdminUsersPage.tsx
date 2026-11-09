import { Fragment, useCallback, useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { api, errorMessage, type AdminUser, type Page, type User } from '../api'
import { useAuth } from '../auth'
import { PenIcon, TrashIcon } from '../components/Icons'
import ProfileFields, { type ProfileData } from '../components/ProfileFields'
import { personName } from '../format'
import { Pager, usePagedSearch } from '../paging'

/**
 * Benutzer (globale Administration): alle Personen über alle Projekte. Nur hier lassen sich Profile ändern
 * (z.B. Email für einen Gast) und Personen löschen. Jeder Zugriff und jede Änderung wird protokolliert.
 */
export default function AdminUsersPage() {
  const { user } = useAuth()
  const [editing, setEditing] = useState<string | null>(null)
  const fetchPage = useCallback(
    (q: string, offset: number) => api.get<Page<AdminUser>>(`/admin/users?q=${encodeURIComponent(q)}&offset=${offset}`),
    [],
  )
  const { query, setQuery, page, setOffset, error, setError, updateItems, reload } = usePagedSearch(fetchPage)
  const users = page?.items ?? null

  async function remove(u: AdminUser) {
    if (!confirm(`${personName(u)} endgültig löschen? Besuche an Events bleiben anonym für die Statistik erhalten.`)) {
      return
    }
    setError('')
    try {
      await api.del(`/admin/users/${u.id}`)
      // Neu laden, damit die Seite wieder voll ist und die Gesamtzahl stimmt
      reload()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="stack">
      <div className="page-header">
        <h1>Benutzer</h1>
        {page && <span className="muted small">{page.total} Benutzer</span>}
      </div>
      <p className="muted small">
        Alle Personen über alle Projekte. Das Öffnen eines Profils wird im Audit-Protokoll festgehalten.
      </p>
      <input placeholder="Suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {error && <p className="error">{error}</p>}
      {users === null && !error && <p className="muted">Laden…</p>}
      {users?.length === 0 && <p className="muted">Keine Benutzer gefunden.</p>}

      {users && users.length > 0 && (
        <table className="table users">
          <thead>
            <tr>
              <th>Name</th>
              <th>Projekte</th>
              <th className="actions-cell">
                <span className="sr-only">Aktionen</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <Fragment key={u.id}>
                <tr className={editing === u.id ? 'open' : undefined}>
                  <td>
                    {personName(u)}
                    {u.guest && <span className="badge">Gast</span>}
                    {u.globalAdmin && <span className="badge success">Global</span>}
                    <br />
                    <span className="muted small">{u.email || 'keine Email'}</span>
                  </td>
                  <td className="small">{u.projects.join(', ') || <span className="muted">–</span>}</td>
                  <td className="actions-cell">
                    <IconButton
                      label={`${personName(u)} bearbeiten`}
                      active={editing === u.id}
                      onClick={() => setEditing(editing === u.id ? null : u.id)}
                    >
                      <PenIcon />
                    </IconButton>
                    {u.id !== user?.id && (
                      <IconButton label={`${personName(u)} löschen`} danger onClick={() => remove(u)}>
                        <TrashIcon />
                      </IconButton>
                    )}
                  </td>
                </tr>
                {editing === u.id && (
                  <tr className="panel-row">
                    <td colSpan={3}>
                      <EditUserPanel
                        userId={u.id}
                        onSaved={(saved) => {
                          updateItems(
                            (list) =>
                              list.map((x) =>
                                x.id === u.id
                                  ? { ...x, firstName: saved.firstName, lastName: saved.lastName, email: saved.email }
                                  : x,
                              ) ?? null,
                          )
                          setEditing(null)
                        }}
                        onCancel={() => setEditing(null)}
                      />
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
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

/** Profil einer Person bearbeiten. Die Email ist nur bei Gästen änderbar. */
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
      .get<User>(`/admin/users/${userId}`)
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
      onSaved(await api.put<User>(`/admin/users/${userId}`, form))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (!target || !form) return error ? <p className="error small">{error}</p> : <p className="muted small">Laden…</p>
  return (
    <form className="stack" onSubmit={submit}>
      {target.guest && (
        <p className="muted small">
          Meldet sich der Gast mit der hier erfassten Email über Google oder Apple an, übernimmt er dieses Konto.
        </p>
      )}
      <ProfileFields value={form} onChange={setForm} email={target.guest ? 'optional' : 'readonly'} />
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
