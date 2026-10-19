import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, type Role, type User } from './api'

interface AuthState {
  user: User | null
  loading: boolean
  setUser: (user: User | null) => void
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      setUser(await api.get<User>('/me'))
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
    setUser(null)
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return (
    <AuthContext.Provider value={{ user, loading, setUser, refresh, logout }}>{children}</AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth ausserhalb von AuthProvider')
  return ctx
}

// Rollenprüfungen für Benutzer und Team-Mitglieder (alles mit einer Rolle)
type HasRole = { role: Role } | null | undefined

/** Administration, Rollen und Funktionen vergeben */
export function isAdmin(user: HasRole): boolean {
  return user?.role === 'admin'
}

/** Berichte, Zeiterfassung, Löhne und Auszahlung: Projektleiter, Admin */
export function isProjectLead(user: HasRole): boolean {
  return user?.role === 'projektleiter' || isAdmin(user)
}

/** Events verwalten und Teilnehmer erfassen: Projektleiter, Mitarbeiter, Admin */
export function isLeader(user: HasRole): boolean {
  return user?.role === 'mitarbeiter' || isProjectLead(user)
}

/** Benutzerliste: Admins (verwalten) und Personen mit Zugriff auf Notfallkontakte (nur ansehen) */
export function canViewUsers(user: User | null): boolean {
  return isAdmin(user) || !!user?.emergencyAccess
}
