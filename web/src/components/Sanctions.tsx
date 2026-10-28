import { useState, type FormEvent } from 'react'
import { errorMessage, type Sanction, type SanctionLevel } from '../api'
import { formatShortDate, sanctionLabels } from '../format'
import { useProject } from '../project'

/** Kurzanzeige in Listen: "Gesperrt" bzw. "Verwarnt", die schwerste zuerst */
export function SanctionBadge({ sanctions }: { sanctions: Sanction[] }) {
  if (sanctions.length === 0) return null
  const s = sanctions[0]
  return (
    <span className={`badge ${s.level === 'sperre' ? 'danger' : 'warn'}`} title={sanctionTitle(s)}>
      {sanctionLabels[s.level]}
    </span>
  )
}

/** Meldung beim Einchecken (Scan und manuell): Stufe, Grund, Projekt und Frist */
export function SanctionNotice({ sanctions }: { sanctions: Sanction[] }) {
  if (sanctions.length === 0) return null
  return (
    <div className={`sanction-notice ${sanctions[0].level}`} role="alert">
      {sanctions.map((s, i) => (
        <p key={i}>
          <strong>{s.level === 'sperre' ? '⛔ Gesperrt' : '⚠ Verwarnt'}</strong>
          {!s.own && <span> (von {s.projectName})</span>}: {s.reason}
          <br />
          <span className="small">
            seit {formatShortDate(s.since.slice(0, 10))}
            {s.until ? `, bis ${formatShortDate(s.until.slice(0, 10))}` : ', unbefristet'}
          </span>
        </p>
      ))}
    </div>
  )
}

function sanctionTitle(s: Sanction): string {
  return `${sanctionLabels[s.level]}${s.own ? '' : ` (${s.projectName})`}: ${s.reason}`
}

/**
 * Verwarnen oder sperren (Projektleiter und Admins), auf der Seite Teilnehmer und am Event. Eine neue Verwarnung oder Sperre ersetzt die bisherige
 * dieses Projekts, die Frist beginnt neu. Geteilte Einträge anderer Projekte sind nur sichtbar.
 */
export function SanctionForm({
  userId,
  name,
  sanctions,
  onSaved,
}: {
  userId: string
  name: string
  sanctions: Sanction[]
  onSaved: () => void
}) {
  const { api } = useProject()
  const own = sanctions.find((s) => s.own)
  const [level, setLevel] = useState<SanctionLevel>(own?.level ?? 'verwarnung')
  const [reason, setReason] = useState(own?.reason ?? '')
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.put(`/members/${userId}/sanction`, { level, reason })
      onSaved()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove() {
    if (!confirm(`Verwarnung bzw. Sperre von ${name} aufheben?`)) return
    try {
      await api.del(`/members/${userId}/sanction`)
      onSaved()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="stack" onSubmit={submit}>
      <SanctionNotice sanctions={sanctions} />
      <label>
        Massnahme
        <select value={level} onChange={(e) => setLevel(e.target.value as SanctionLevel)}>
          <option value="verwarnung">Verwarnen (Check-in mit Meldung)</option>
          <option value="sperre">Sperren (kein Check-in, keine Anmeldung)</option>
        </select>
      </label>
      <label>
        Grund
        <input required value={reason} onChange={(e) => setReason(e.target.value)} placeholder="z.B. Regeln missachtet" />
      </label>
      <p className="muted small">
        Wird beim Einchecken angezeigt. Fristen legt der Projekt-Admin unter Administration fest.
      </p>
      {error && <p className="error">{error}</p>}
      <div className="actions">
        {own && (
          <button type="button" className="danger" onClick={remove}>
            Aufheben
          </button>
        )}
        <span className="grow" />
        <button type="submit" className="primary">
          {own ? 'Ersetzen' : 'Speichern'}
        </button>
      </div>
    </form>
  )
}
