import { BrowserRouter, Navigate, NavLink, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import type { Role } from './api'
import { AuthProvider, isLeader, useAuth } from './auth'
import { roleLabels } from './format'
import AdminLocationsPage from './pages/AdminLocationsPage'
import AdminUsersPage from './pages/AdminUsersPage'
import EventDetailPage from './pages/EventDetailPage'
import EventFormPage from './pages/EventFormPage'
import EventsPage from './pages/EventsPage'
import LoginPage from './pages/LoginPage'
import ProfilePage from './pages/ProfilePage'
import QrPage from './pages/QrPage'
import ScanPage from './pages/ScanPage'
import TimeEntriesPage from './pages/TimeEntriesPage'

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
            <Route element={<RequireRole roles={['projektleiter', 'admin']} />}>
              <Route path="events/neu" element={<EventFormPage />} />
              <Route path="events/:id/bearbeiten" element={<EventFormPage />} />
              <Route path="events/:id/scan" element={<ScanPage />} />
              <Route path="zeiten" element={<TimeEntriesPage />} />
            </Route>
            <Route element={<RequireRole roles={['admin']} />}>
              <Route path="admin/benutzer" element={<AdminUsersPage />} />
              <Route path="admin/orte" element={<AdminLocationsPage />} />
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
          <img src="/icon.svg" alt="" width={28} height={28} />
          Midnightsports
        </NavLink>
        <span className="muted small">
          {user.firstName || user.email} · {roleLabels[user.role]}
        </span>
      </header>
      {user.profileComplete && (
        <nav className="nav">
          <NavLink to="/events">Events</NavLink>
          <NavLink to="/qr">Mein QR-Code</NavLink>
          {isLeader(user) && <NavLink to="/zeiten">Zeiterfassung</NavLink>}
          {user.role === 'admin' && <NavLink to="/admin/benutzer">Benutzer</NavLink>}
          {user.role === 'admin' && <NavLink to="/admin/orte">Orte</NavLink>}
          <NavLink to="/profil">Profil</NavLink>
          <button className="link" onClick={logout}>
            Abmelden
          </button>
        </nav>
      )}
      <main className="container">
        <Outlet />
      </main>
    </>
  )
}
