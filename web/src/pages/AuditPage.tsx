import { useCallback, useEffect, useState } from 'react'
import { api, errorMessage, type AdminProject, type AuditRecord } from '../api'
import { auditLabels } from '../format'

/** Gruppen für den Filter nach Aktion (Präfix der Aktion im Audit-Protokoll) */
const actionGroups: Record<string, string> = {
  '': 'Alle Aktionen',
  notfallkontakt: 'Notfallkontakte',
  profil: 'Profile',
  sanktion: 'Verwarnungen und Sperren',
  mitglied: 'Mitgliedschaften',
  teilen: 'Teilen zwischen Projekten',
  export: 'Exporte',
  benutzer: 'Gelöschte Benutzer',
  konto: 'Gelöschte Konten',
  projekt: 'Projekte',
  gast: 'Gäste',
  notfallzugriff: 'Zugriffsrechte',
}

/**
 * Audit-Protokoll (globale Administration): wer hat wann was mit wessen Daten gemacht. Neuste zuerst, in Seiten
 * zu 100 Einträgen. Namen gelöschter Personen erscheinen als «Gelöschtes Konto».
 */
export default function AuditPage() {
  const [records, setRecords] = useState<AuditRecord[]>([])
  const [projects, setProjects] = useState<AdminProject[]>([])
  const [filter, setFilter] = useState({ q: '', project: '', action: '' })
  const [more, setMore] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(
    async (before?: number) => {
      const params = new URLSearchParams(filter)
      if (before) params.set('before', String(before))
      try {
        const page = await api.get<AuditRecord[]>(`/admin/audit?${params}`)
        setRecords((list) => (before ? [...list, ...page] : page))
        setMore(page.length === 100)
        setError('')
      } catch (err) {
        setError(errorMessage(err))
      }
    },
    [filter],
  )

  useEffect(() => {
    // Suchbegriff erst nach einer kurzen Pause laden
    const t = setTimeout(() => load(), 250)
    return () => clearTimeout(t)
  }, [load])

  useEffect(() => {
    api
      .get<AdminProject[]>('/admin/projects')
      .then(setProjects)
      .catch(() => setProjects([]))
  }, [])

  return (
    <div className="stack">
      <h1>Audit-Protokoll</h1>
      <p className="muted small">
        Zugriffe auf Notfallkontakte und Profile, Änderungen an Personendaten und Berechtigungen, Löschungen, Exporte,
        Verwarnungen und Sperren sowie das Teilen zwischen Projekten.
      </p>
      <div className="row">
        <input
          placeholder="Person (handelnd oder betroffen)"
          value={filter.q}
          onChange={(e) => setFilter({ ...filter, q: e.target.value })}
        />
        <select
          aria-label="Projekt"
          value={filter.project}
          onChange={(e) => setFilter({ ...filter, project: e.target.value })}
        >
          <option value="">Alle Projekte</option>
          {projects.map((p) => (
            <option key={p.slug} value={p.slug}>
              {p.name}
            </option>
          ))}
        </select>
        <select
          aria-label="Aktion"
          value={filter.action}
          onChange={(e) => setFilter({ ...filter, action: e.target.value })}
        >
          {Object.entries(actionGroups).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>
      {error && <p className="error">{error}</p>}
      {/* Breite Tabelle: auf dem Handy innerhalb des Rahmens scrollen, nicht die ganze Seite */}
      <div className="table-scroll">
        <table className="table audit">
          <thead>
            <tr>
              <th>Zeit</th>
              <th>Aktion</th>
              <th>Wer</th>
              <th>Betroffen</th>
              <th>Projekt</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody>
            {records.map((r) => (
              <tr key={r.id}>
                <td className="small">
                  {new Date(r.at).toLocaleString('de-CH', {
                    dateStyle: 'short',
                    timeStyle: 'short',
                  })}
                </td>
                <td>{auditLabels[r.action] ?? r.action}</td>
                <td>{r.actorName || '–'}</td>
                <td>{r.targetName || '–'}</td>
                <td className="small">{r.projectName ?? '–'}</td>
                <td className="small muted">{formatDetails(r.details)}</td>
              </tr>
            ))}
            {records.length === 0 && (
              <tr>
                <td colSpan={6} className="muted">
                  Keine Einträge.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {more && (
        <div className="actions">
          <button onClick={() => load(records[records.length - 1].id)}>Ältere laden</button>
        </div>
      )}
    </div>
  )
}

/** {"vorher":"teilnehmer","nachher":"admin"} → "vorher: teilnehmer, nachher: admin" */
function formatDetails(details: Record<string, unknown>): string {
  return Object.entries(details)
    .map(([k, v]) => `${k}: ${Array.isArray(v) ? v.join(', ') : v === null ? '–' : String(v)}`)
    .join(' · ')
}
