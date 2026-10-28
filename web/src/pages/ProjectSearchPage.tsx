import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, errorMessage, type ProjectSearchResult } from '../api'
import { projectLogo } from '../project'

/** Projekte suchen (Name, Region, Beschreibung) und sich für ein neues Projekt anmelden. */
export default function ProjectSearchPage() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<ProjectSearchResult[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    // Antworten zu einem veralteten Suchbegriff ignorieren
    let current = true
    const t = setTimeout(() => {
      api
        .get<ProjectSearchResult[]>(`/projects?q=${encodeURIComponent(query)}`)
        .then((list) => current && setResults(list))
        .catch((err) => current && setError(errorMessage(err)))
    }, 250)
    return () => {
      current = false
      clearTimeout(t)
    }
  }, [query])

  return (
    <div className="narrow stack">
      <h1>Projekte suchen</h1>
      <input
        type="search"
        autoFocus
        placeholder="Name oder Ort, z.B. Aadorf"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      {error && <p className="error">{error}</p>}
      {results?.length === 0 && <p className="muted">Keine Projekte gefunden.</p>}
      <ul className="list">
        {results?.map((p) => (
          <li key={p.slug} className="project-result">
            <img src={projectLogo(p)} alt="" height={32} />
            <span className="grow">
              <strong>{p.name}</strong>
              <br />
              <span className="muted small">{p.description || p.region}</span>
            </span>
            {p.member ? (
              <Link to={`/p/${p.slug}/events`} className="button small">
                Öffnen
              </Link>
            ) : (
              <Link to={`/projekte/${p.slug}`} className="button small primary">
                Anmelden
              </Link>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
