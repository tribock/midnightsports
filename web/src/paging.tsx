import { useCallback, useEffect, useState } from 'react'
import { errorMessage, type Page } from './api'
import { pageRange } from './format'

/**
 * Suche mit Seiten: lädt 250 ms nach der letzten Eingabe, ignoriert Antworten zu veralteten Suchbegriffen oder Seiten.
 * Ein neuer Suchbegriff beginnt wieder auf Seite 1. fetchPage muss stabil sein (useCallback).
 */
export function usePagedSearch<T>(fetchPage: (query: string, offset: number) => Promise<Page<T>>) {
  const [query, setQueryState] = useState('')
  const [offset, setOffset] = useState(0)
  const [page, setPage] = useState<Page<T> | null>(null)
  const [error, setError] = useState('')
  const [version, setVersion] = useState(0)

  useEffect(() => {
    let current = true
    const t = setTimeout(() => {
      fetchPage(query, offset)
        .then((p) => {
          if (!current) return
          // Hinter der letzten Seite (z.B. nach dem Löschen) zurück auf Seite 1
          if (p.items.length === 0 && offset > 0) setOffset(0)
          else setPage(p)
          setError('')
        })
        .catch((err) => current && setError(errorMessage(err)))
    }, 250)
    return () => {
      current = false
      clearTimeout(t)
    }
  }, [fetchPage, query, offset, version])

  const setQuery = useCallback((q: string) => {
    setQueryState(q)
    setOffset(0)
  }, [])

  /** Einträge der aktuellen Seite ändern, ohne neu zu laden (z.B. nach dem Speichern einer Rolle) */
  const updateItems = useCallback((change: (items: T[]) => T[]) => {
    setPage((p) => p && { ...p, items: change(p.items) })
  }, [])

  const reload = useCallback(() => setVersion((v) => v + 1), [])

  return { query, setQuery, page, setOffset, error, setError, updateItems, reload }
}

/** Blättern: "51–100 von 230", Zurück und Weiter. Ohne weitere Seiten nur die Anzahl. */
export function Pager({ page, onChange }: { page: Page<unknown>; onChange: (offset: number) => void }) {
  const hasPrev = page.offset > 0
  const hasNext = page.offset + page.items.length < page.total
  return (
    <div className="pager">
      <span className="muted small">{pageRange(page)}</span>
      {(hasPrev || hasNext) && (
        <span className="actions">
          <button className="small" disabled={!hasPrev} onClick={() => onChange(Math.max(0, page.offset - page.limit))}>
            Zurück
          </button>
          <button className="small" disabled={!hasNext} onClick={() => onChange(page.offset + page.limit)}>
            Weiter
          </button>
        </span>
      )}
    </div>
  )
}
