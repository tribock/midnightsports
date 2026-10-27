import type { Sanction } from '../api'
import { formatShortDate, sanctionLabels } from '../format'

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
