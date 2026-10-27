import { useEffect, useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { api, errorMessage, type PublicProject, type UserStatus } from '../api'
import { useAuth } from '../auth'
import StatusSelect from '../components/StatusSelect'
import { projectLogo } from '../project'

/**
 * Anmeldung bei einem Projekt (aus der Suche oder über den Projekt-Link /?projekt=…).
 * Bereits angemeldete Mitglieder gelangen direkt ins Projekt.
 */
export default function ProjectJoinPage() {
  const { slug = '' } = useParams()
  const { user, refresh } = useAuth()
  const navigate = useNavigate()
  const [project, setProject] = useState<PublicProject | null>(null)
  const [status, setStatus] = useState<UserStatus | ''>('')
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .get<PublicProject>(`/public/projects/${slug}`)
      .then(setProject)
      .catch((err) => setError(errorMessage(err)))
  }, [slug])

  if (user!.projects.some((p) => p.slug === slug)) return <Navigate to={`/p/${slug}/events`} replace />

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.post(`/projects/${slug}/join`, { status })
      await refresh()
      navigate(`/p/${slug}/events`)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (!project) return error ? <p className="error">{error}</p> : <p className="muted">Laden…</p>
  return (
    <div className="narrow stack center">
      <img src={projectLogo(project)} alt={project.name} className="project-logo" />
      <h1>{project.name}</h1>
      <p className="muted">
        Melde dich an, um dich für Events von {project.name} anzumelden und am Eingang einzuchecken.
      </p>
      <form className="card stack" onSubmit={submit}>
        <StatusSelect value={status} onChange={setStatus} region={project.region} label="Ich bin" />
        {error && <p className="error">{error}</p>}
        <button type="submit" className="primary">
          Bei {project.name} anmelden
        </button>
      </form>
      <Link to="/projekte" className="muted">
        Andere Projekte suchen
      </Link>
    </div>
  )
}
