import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, type User } from './api'

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

export function isLeader(user: User | null): boolean {
  return user?.role === 'projektleiter' || user?.role === 'admin'
}
