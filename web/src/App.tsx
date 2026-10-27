import { lazy, Suspense, useEffect, useState } from 'react'
import { BrowserRouter, Navigate, NavLink, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import { api, type BuildInfo, type Role } from './api'
import { AuthProvider, canViewUsers, isAdmin, isProjectLead, useAuth } from './auth'
import { roleLabels } from './format'
import AdministrationPage from './pages/AdministrationPage'
import AdminUsersPage from './pages/AdminUsersPage'
import EventDetailPage from './pages/EventDetailPage'
import EventFormPage from './pages/EventFormPage'
import EventsPage from './pages/EventsPage'
import LoginPage from './pages/LoginPage'
import PayoutPage from './pages/PayoutPage'
import ProfilePage from './pages/ProfilePage'
import TeamPage from './pages/TeamPage'
import TimeEntriesPage from './pages/TimeEntriesPage'

// Seiten mit grossen Bibliotheken (QR-Code erzeugen bzw. scannen) erst bei Bedarf laden
const QrPage = lazy(() => import('./pages/QrPage'))
const ScanPage = lazy(() => import('./pages/ScanPage'))

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<RequireAuth />}>
            <Route index element={<Navigate to="/events" replace />} />
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
            <Route element={<RequireUsersAccess />}>
              <Route path="admin/benutzer" element={<AdminUsersPage />} />
            </Route>
            <Route element={<RequireRole roles={['admin']} />}>
              <Route path="admin/orte" element={<Navigate to="/administration" replace />} />
              <Route path="administration" element={<AdministrationPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/events" replace />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

function RequireAuth() {
  const { user, loading } = useAuth()
  const location = useLocation()

  if (loading) return <p className="center muted">Laden…</p>
  if (!user) return <Navigate to="/login" replace />
  // Ohne vollständiges Profil ist nur die Profilseite erreichbar
  if (!user.profileComplete && location.pathname !== '/profil') return <Navigate to="/profil" replace />

  return <Layout />
}

function RequireUsersAccess() {
  const { user } = useAuth()
  return canViewUsers(user) ? <Outlet /> : <Navigate to="/events" replace />
}

function RequireRole({ roles }: { roles: Role[] }) {
  const { user } = useAuth()
  return user && roles.includes(user.role) ? <Outlet /> : <Navigate to="/events" replace />
}

function Layout() {
  const { user, logout } = useAuth()
  if (!user) return null

  return (
    <>
      <header className="topbar">
        <NavLink to="/events" className="brand">
          <img src="/logo-white.svg" alt="Midnightsports" width={158} height={28} />
        </NavLink>
        <span className="muted small">
          {user.firstName || user.email} · {roleLabels[user.role]}
        </span>
      </header>
      {user.profileComplete && (
        <nav className="nav">
          <NavLink to="/events">Events</NavLink>
          <NavLink to="/qr">Mein QR-Code</NavLink>
          {isProjectLead(user) && <NavLink to="/zeiten">Zeiterfassung</NavLink>}
          {isProjectLead(user) && <NavLink to="/team">Team</NavLink>}
          {canViewUsers(user) && <NavLink to="/admin/benutzer">Benutzer</NavLink>}
          {isAdmin(user) && <NavLink to="/administration">Administration</NavLink>}
          <NavLink to="/profil">Profil</NavLink>
          <button className="link" onClick={logout}>
            Abmelden
          </button>
        </nav>
      )}
      <main className="container">
        <Suspense fallback={<p className="muted">Laden…</p>}>
          <Outlet />
        </Suspense>
      </main>
      <VersionFooter />
    </>
  )
}

function VersionFooter() {
  const [build, setBuild] = useState<BuildInfo | null>(null)

  useEffect(() => {
    api.get<BuildInfo>('/version').then(setBuild).catch(() => setBuild(null))
  }, [])

  if (!build) return null
  return (
    <footer className="footer muted small" title={`Build: ${build.date}`}>
      Version {build.version}
    </footer>
  )
}
