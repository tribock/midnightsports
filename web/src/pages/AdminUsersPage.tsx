import { useEffect, useState } from 'react'
import { api, errorMessage, type Role, type UserSummary } from '../api'
import { useAuth } from '../auth'
import { roleLabels } from '../format'

export default function AdminUsersPage() {
  const { user } = useAuth()
  const [query, setQuery] = useState('')
  const [users, setUsers] = useState<UserSummary[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    const t = setTimeout(() => {
      api
        .get<UserSummary[]>(`/users?q=${encodeURIComponent(query)}`)
        .then(setUsers)
        .catch((err) => setError(errorMessage(err)))
    }, 250)
    return () => clearTimeout(t)
  }, [query])

  async function setRole(u: UserSummary, role: Role) {
    setError('')
    try {
      await api.put(`/users/${u.id}/role`, { role })
      setUsers((list) => list.map((x) => (x.id === u.id ? { ...x, role } : x)))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="stack">
      <h1>Benutzer</h1>
      <input placeholder="Suchen (Name oder Email)" value={query} onChange={(e) => setQuery(e.target.value)} />
      {error && <p className="error">{error}</p>}
      <ul className="list">
        {users.map((u) => (
          <li key={u.id}>
            <span className="grow">
              {u.firstName || u.lastName ? `${u.firstName} ${u.lastName}` : '(Profil unvollständig)'}
              <br />
              <span className="muted small">{u.email}</span>
            </span>
            <select
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
          </li>
        ))}
      </ul>
    </div>
  )
}
