import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { Navigate, useParams } from 'react-router-dom'
import { apiWithPrefix, ApiError, errorMessage, type Api, type Member, type Project } from './api'

interface ProjectState {
  member: Member
  project: Project
  /** API mit Präfix /p/{slug} */
  api: Api
  /** Basis für Links im Projekt, z.B. "/p/aadorf" */
  base: string
  reload: () => Promise<void>
}

const ProjectContext = createContext<ProjectState | null>(null)

/** Lädt Projekt und Mitgliedschaft für alle Seiten unter /p/:slug. Nicht angemeldet → Seite zum Anmelden beim Projekt. */
export function ProjectProvider({ children }: { children: ReactNode }) {
  const { slug = '' } = useParams()
  const [member, setMember] = useState<Member | null>(null)
  const [error, setError] = useState<{ status: number; message: string } | null>(null)
  const scoped = useMemo(() => apiWithPrefix(`/p/${slug}`), [slug])

  const reload = useCallback(async () => {
    try {
      setMember(await scoped.get<Member>('/me'))
      setError(null)
    } catch (err) {
      setError({ status: err instanceof ApiError ? err.status : 0, message: errorMessage(err) })
    }
  }, [scoped])

  useEffect(() => {
    setMember(null)
    reload()
    // Zuletzt gewähltes Projekt für die Startseite merken
    try {
      localStorage.setItem('ms_last_project', slug)
    } catch {
      // nur eine Bequemlichkeit
    }
  }, [slug, reload])

  if (error?.status === 403) return <Navigate to={`/projekte/${slug}`} replace />
  if (error) return <p className="error">{error.message}</p>
  if (!member || member.project.slug !== slug) return <p className="muted">Laden…</p>
  return (
    <ProjectContext.Provider value={{ member, project: member.project, api: scoped, base: `/p/${slug}`, reload }}>
      {children}
    </ProjectContext.Provider>
  )
}

export function useProject(): ProjectState {
  const ctx = useContext(ProjectContext)
  if (!ctx) throw new Error('useProject ausserhalb von ProjectProvider')
  return ctx
}

/** Projekt, falls die Seite in einem Projekt liegt (Kopfzeile, Druck) */
export function useOptionalProject(): ProjectState | null {
  return useContext(ProjectContext)
}

/** Logo eines Projekts: hochgeladenes Logo oder Standardlogo (variant white für dunklen Grund) */
export function projectLogo(
  p: { slug: string; hasLogo: boolean } | null | undefined,
  variant: 'color' | 'white' = 'color',
) {
  if (p?.hasLogo) return `/api/public/projects/${p.slug}/logo`
  return variant === 'white' ? '/logo-white.svg' : '/logo.svg'
}
