import { lazy, Suspense, useEffect, useState } from 'react'
import {
  BrowserRouter,
  Navigate,
  NavLink,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useSearchParams,
} from 'react-router-dom'
import { api, type BuildInfo, type Role } from './api'
import { afterLogin, AuthProvider, canViewMembers, isAdmin, isProjectLead, rememberAfterLogin, useAuth } from './auth'
import { roleLabels } from './format'
import AdminProjectsPage from './pages/AdminProjectsPage'
import AdminUsersPage from './pages/AdminUsersPage'
import AuditPage from './pages/AuditPage'
import EventDetailPage from './pages/EventDetailPage'
import EventFormPage from './pages/EventFormPage'
import EventsPage from './pages/EventsPage'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import MembersPage from './pages/MembersPage'
import PayoutPage from './pages/PayoutPage'
import ProfilePage from './pages/ProfilePage'
import ProjectJoinPage from './pages/ProjectJoinPage'
import ProjectSearchPage from './pages/ProjectSearchPage'
import TeamPage from './pages/TeamPage'
import TimeEntriesPage from './pages/TimeEntriesPage'
import { projectLogo, ProjectProvider, useOptionalProject, useProject } from './project'

// Seiten mit grossen Bibliotheken (QR-Code erzeugen bzw. scannen) erst bei Bedarf laden
const QrPage = lazy(() => import('./pages/QrPage'))
const ScanPage = lazy(() => import('./pages/ScanPage'))
const AdministrationPage = lazy(() => import('./pages/AdministrationPage'))

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<RequireAuth />}>
            {/* Ohne Projekt: Startseite, Projekte suchen, eigenes Profil, globale Administration */}
            <Route element={<Layout />}>
              <Route index element={<Start />} />
              <Route path="projekte" element={<ProjectSearchPage />} />
              <Route path="projekte/:slug" element={<ProjectJoinPage />} />
              <Route path="qr" element={<QrPage />} />
              <Route path="profil" element={<ProfilePage />} />
              <Route element={<RequireGlobalAdmin />}>
                <Route path="admin/projekte" element={<AdminProjectsPage />} />
                <Route path="admin/benutzer" element={<AdminUsersPage />} />
                <Route path="admin/audit" element={<AuditPage />} />
              </Route>
            </Route>
            {/* Im Projekt: Rollen gelten pro Projekt */}
            <Route
              path="p/:slug"
              element={
                <ProjectProvider>
                  <Layout />
                </ProjectProvider>
              }
            >
              <Route index element={<Navigate to="events" replace />} />
              <Route path="events" element={<EventsPage />} />
              <Route path="events/:id" element={<EventDetailPage />} />
              <Route path="qr" element={<QrPage />} />
              <Route path="profil" element={<ProfilePage />} />
              <Route element={<RequireRole roles={['projektleiter', 'mitarbeiter', 'admin']} />}>
                <Route path="events/:id/scan" element={<ScanPage />} />
              </Route>
              <Route element={<RequireRole roles={['projektleiter', 'admin']} />}>
                <Route path="events/neu" element={<EventFormPage />} />
                <Route path="events/:id/bearbeiten" element={<EventFormPage />} />
                <Route path="zeiten" element={<TimeEntriesPage />} />
                <Route path="team" element={<TeamPage />} />
                <Route path="events/:id/auszahlung" element={<PayoutPage />} />
              </Route>
              <Route element={<RequireMembersAccess />}>
                <Route path="teilnehmer" element={<MembersPage />} />
              </Route>
              <Route element={<RequireRole roles={['admin']} />}>
                <Route path="administration" element={<AdministrationPage />} />
              </Route>
              <Route path="*" element={<Navigate to="events" replace />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

/** Ziel nach dem Login: der Projekt-Link /?projekt=aadorf führt zur Anmeldung bei diesem Projekt. */
function loginTarget(pathname: string, search: string): string {
  const slug = new URLSearchParams(search).get('projekt')
  if (pathname === '/' && slug) return `/projekte/${encodeURIComponent(slug)}`
  return pathname + search
}

function RequireAuth() {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) return <p className="center muted">Laden…</p>
  if (!user) {
    const target = loginTarget(location.pathname, location.search)
    if (target !== '/') rememberAfterLogin(target)
    return <Navigate to="/login" replace />
  }
  // Ohne vollständiges Profil ist nur die Profilseite erreichbar. Das Ziel (z.B. Projekt-Link) bleibt gemerkt.
  // Globale Admins verwalten Projekte auch ohne Profil (Geburtsdatum usw. braucht es nur für Projekte).
  const adminPage = user.globalAdmin && location.pathname.startsWith('/admin/')
  if (!user.profileComplete && !adminPage) {
    if (location.pathname !== '/profil') {
      const target = loginTarget(location.pathname, location.search)
      if (target !== '/') rememberAfterLogin(target)
      return <Navigate to="/profil" replace />
    }
    return <Outlet />
  }
  // Zurück vom Login bei Google oder Apple: zum gemerkten Ziel
  const target = afterLogin(true)
  if (target && target !== location.pathname + location.search) return <Navigate to={target} replace />
  return <Outlet />
}

/** Startseite: Projekt-Link (/?projekt=…) oder Projektauswahl */
function Start() {
  const [params] = useSearchParams()
  const slug = params.get('projekt')
  if (slug) return <Navigate to={`/projekte/${encodeURIComponent(slug)}`} replace />
  return <HomePage />
}

function RequireGlobalAdmin() {
  const { user } = useAuth()
  return user?.globalAdmin ? <Outlet /> : <Navigate to="/" replace />
}

function RequireMembersAccess() {
  const { member, base } = useProject()
  return canViewMembers(member) ? <Outlet /> : <Navigate to={`${base}/events`} replace />
}

function RequireRole({ roles }: { roles: Role[] }) {
  const { member, base } = useProject()
  return roles.includes(member.role) ? <Outlet /> : <Navigate to={`${base}/events`} replace />
}

/** Kopfzeile mit Projektwahl und Navigation. Im Projekt mit dessen Logo und Seiten, sonst die globalen Seiten. */
function Layout() {
  const { user, logout } = useAuth()
  const ctx = useOptionalProject()
  if (!user) return null
  const base = ctx?.base

  return (
    <>
      <header className="topbar">
        <NavLink to={base ? `${base}/events` : '/'} className="brand">
          <img
            src={projectLogo(ctx?.project, 'white')}
            alt={ctx?.project.name ?? 'Midnightsports'}
            className={ctx?.project.hasLogo ? 'brand-logo custom' : 'brand-logo'}
            height={28}
          />
        </NavLink>
        <ProjectSwitcher />
        <span className="muted small topbar-user">
          {user.firstName || user.email}
          {ctx ? ` · ${roleLabels[ctx.member.role]}` : user.globalAdmin ? ' · Global-Admin' : ''}
        </span>
      </header>
      {(user.profileComplete || user.globalAdmin) && (
        <nav className="nav">
          {ctx && base ? (
            <>
              <NavLink to={`${base}/events`}>Events</NavLink>
              <NavLink to={`${base}/qr`}>Mein QR-Code</NavLink>
              {isProjectLead(ctx.member) && <NavLink to={`${base}/zeiten`}>Zeiterfassung</NavLink>}
              {isProjectLead(ctx.member) && <NavLink to={`${base}/team`}>Team</NavLink>}
              {canViewMembers(ctx.member) && <NavLink to={`${base}/teilnehmer`}>Teilnehmer</NavLink>}
              {isAdmin(ctx.member) && <NavLink to={`${base}/administration`}>Administration</NavLink>}
              <NavLink to={`${base}/profil`}>Profil</NavLink>
            </>
          ) : (
            <>
              <NavLink to="/" end>
                Übersicht
              </NavLink>
              <NavLink to="/qr">Mein QR-Code</NavLink>
              <NavLink to="/profil">Profil</NavLink>
              {user.globalAdmin && (
                <>
                  <NavLink to="/admin/projekte">Alle Projekte</NavLink>
                  <NavLink to="/admin/benutzer">Benutzer</NavLink>
                  <NavLink to="/admin/audit">Audit</NavLink>
                </>
              )}
            </>
          )}
          <button className="link" onClick={logout}>
            Abmelden
          </button>
        </nav>
      )}
      <main className="container">
        {ctx?.project.pending && (
          <p className="notice">
            Dieses Projekt wartet auf die Einwilligung eines Partnerprojekts und ist nur für globale Admins sichtbar.
          </p>
        )}
        <Suspense fallback={<p className="muted">Laden…</p>}>
          <Outlet />
        </Suspense>
      </main>
      <VersionFooter />
    </>
  )
}

/** Projekt wechseln: eigene Projekte, Suche, für globale Admins die Administration */
function ProjectSwitcher() {
  const { user } = useAuth()
  const ctx = useOptionalProject()
  const navigate = useNavigate()
  if (!user?.profileComplete) return null
  const current = ctx?.project.slug ?? ''
  const own = user.projects.some((p) => p.slug === current)

  return (
    <select
      className="project-switcher"
      aria-label="Projekt wählen"
      value={current}
      onChange={(e) => {
        const v = e.target.value
        navigate(v === '__suche' ? '/projekte' : v === '' ? '/' : `/p/${v}/events`)
      }}
    >
      <option value="">Übersicht</option>
      {user.projects.map((p) => (
        <option key={p.slug} value={p.slug}>
          {p.name}
        </option>
      ))}
      {ctx && !own && <option value={current}>{ctx.project.name}</option>}
      <option value="__suche">Nach Projekten suchen…</option>
    </select>
  )
}

function VersionFooter() {
  const [build, setBuild] = useState<BuildInfo | null>(null)

  useEffect(() => {
    api
      .get<BuildInfo>('/version')
      .then(setBuild)
      .catch(() => setBuild(null))
  }, [])

  if (!build) return null
  return (
    <footer className="footer muted small" title={`Build: ${build.date}`}>
      Version {build.version}
    </footer>
  )
}
