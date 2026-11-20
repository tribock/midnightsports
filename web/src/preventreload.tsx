import { useEffect } from 'react'

/**
 * Warnt beim Neuladen oder Schliessen der Seite, solange dirty gilt. Der Browser zeigt seinen eigenen Text,
 * Safari auf dem iPhone meist gar nichts: nur ein Sicherheitsnetz.
 */
export function useUnsavedWarning(dirty: boolean) {
  useEffect(() => {
    if (!dirty) return
    const warn = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])
}