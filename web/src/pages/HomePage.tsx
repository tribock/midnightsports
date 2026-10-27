import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { roleLabels } from '../format'

/** zuletzt geöffnetes Projekt (ProjectProvider merkt es sich) */
function lastProject(): string | null {
  try {
    return localStorage.getItem('ms_last_project')
  } catch {
    return null
  }
}

/**
 * Startseite nach dem Login: Projekt aus den eigenen Projekten wählen oder nach Projekten suchen.
 * Neue Personen ohne Projekt landen direkt in der Suche.
 */
export default function HomePage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const projects = user!.projects
  const preferred = projects.find((p) => p.slug === lastProject()) ?? projects[0]
  const [slug, setSlug] = useState(preferred?.slug ?? '')

  if (projects.length === 0 && !user!.globalAdmin) return <Navigate to="/projekte" replace />

  return (
    <div className="narrow stack">
      <h1>Willkommen{user!.firstName ? `, ${user!.firstName}` : ''}</h1>
      {projects.length > 0 ? (
        <form
          className="card stack"
          onSubmit={(e) => {
            e.preventDefault()
            navigate(`/p/${slug}/events`)
          }}
        >
          <label>
            Projekt
            <select value={slug} onChange={(e) => setSlug(e.target.value)}>
              {projects.map((p) => (
                <option key={p.slug} value={p.slug}>
                  {p.name} · {roleLabels[p.role]}
                  {p.pending ? ' (wartet auf Einwilligung)' : ''}
                </option>
              ))}
            </select>
          </label>
          <button type="submit" className="primary">
            Öffnen
          </button>
        </form>
      ) : (
        <p className="muted">Du bist noch bei keinem Projekt angemeldet.</p>
      )}
      <Link to="/projekte" className="button">
        Nach Projekten suchen
      </Link>
      {user!.globalAdmin && (
        <Link to="/admin/projekte" className="button">
          Globale Administration
        </Link>
      )}
    </div>
  )
}
