import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, type Me, type Role } from './api'

interface AuthState {
  user: Me | null
  loading: boolean
  setUser: (user: Me | null) => void
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<Me | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      setUser(await api.get<Me>('/me'))
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 401)) {
        console.error(err)
      }
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    await api.post('/auth/logout')
    // Gemerktes Ziel nicht an die nächste Person auf demselben Gerät weitergeben
    afterLogin(true)
    setUser(null)
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return <AuthContext.Provider value={{ user, loading, setUser, refresh, logout }}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth ausserhalb von AuthProvider')
  return ctx
}

// Rollenprüfungen im Projekt (Mitglied, Team-Mitglied, alles mit einer Rolle)
type HasRole = { role: Role } | null | undefined

/** Administration des Projekts, Rollen und Funktionen vergeben: Projekt-Admin (und globale Admins) */
export function isAdmin(member: HasRole): boolean {
  return member?.role === 'admin'
}

/** Events erfassen, Berichte, Zeiterfassung, Löhne, Auszahlung, Verwarnungen: Projektleiter, Admin */
export function isProjectLead(member: HasRole): boolean {
  return member?.role === 'projektleiter' || isAdmin(member)
}

/** Teilnehmer erfassen und einchecken: Projektleiter, Mitarbeiter, Admin */
export function isLeader(member: HasRole): boolean {
  return member?.role === 'mitarbeiter' || isProjectLead(member)
}

/** Teilnehmerliste: Projektleiter und Admins (verwalten) sowie Personen mit Zugriff auf Notfallkontakte */
export function canViewMembers(member: (HasRole & { emergencyAccess?: boolean }) | null | undefined): boolean {
  return isProjectLead(member) || !!member?.emergencyAccess
}

// Ziel nach dem Login (z.B. der Projekt-Link), überlebt die Weiterleitung zu Google oder Apple
const afterLoginKey = 'ms_after_login'

export function rememberAfterLogin(path: string) {
  try {
    sessionStorage.setItem(afterLoginKey, path)
  } catch {
    // ohne Speicher landet man nach dem Login auf der Startseite
  }
}

/** Liest das gemerkte Ziel. take = true entfernt es (nach dem Login). */
export function afterLogin(take = false): string | null {
  try {
    const path = sessionStorage.getItem(afterLoginKey)
    if (take) sessionStorage.removeItem(afterLoginKey)
    // Nur Pfade dieser App, keine fremden Adressen
    return path && path.startsWith('/') && !path.startsWith('//') ? path : null
  } catch {
    return null
  }
}
