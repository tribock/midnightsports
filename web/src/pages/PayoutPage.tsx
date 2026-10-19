import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, errorMessage, type Event, type PayoutEntry } from '../api'
import EventInfo from '../components/EventInfo'
import PrintSheet, { printDocument } from '../components/PrintSheet'
import SignaturePad from '../components/SignaturePad'
import { formatCHF, formatDate, formatShortDate } from '../format'

/**
 * Auszahlung am Event (Projektleiter und Admins): alle eingeteilten Personen mit Betrag.
 * Jede Person bestätigt den Empfang mit ihrer Unterschrift. Gedruckt wird ein Spesenblatt (PayoutSheet).
 */
export default function PayoutPage() {
  const { id } = useParams()
  const [event, setEvent] = useState<Event | null>(null)
  const [entries, setEntries] = useState<PayoutEntry[]>([])
  const [signing, setSigning] = useState<PayoutEntry | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<PayoutEntry[]>(`/events/${id}/payout`)
      .then(setEntries)
      .catch((err) => setError(errorMessage(err)))
  }, [id])

  useEffect(() => {
    api
      .get<Event>(`/events/${id}`)
      .then(setEvent)
      .catch((err) => setError(errorMessage(err)))
    load()
  }, [id, load])

  async function unsign(e: PayoutEntry) {
    if (!confirm(`Unterschrift von ${e.name} entfernen?`)) return
    try {
      await api.del(`/events/${id}/payout/${e.userId}`)
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (!event) return error ? <p className="error">{error}</p> : <p className="muted">Laden…</p>

  const paid = entries.filter((e) => !e.paidByTime)
  const byTime = entries.filter((e) => e.paidByTime)
  const total = paid.reduce((sum, e) => sum + (e.salary ?? 0), 0)
  const open = paid.filter((e) => !e.signedAt).length

  return (
    <div className="stack">
      <Link to={`/events/${event.id}`} className="muted">
        ← Zurück zum Event
      </Link>
      <div className="page-header">
        <h1>Auszahlung</h1>
        <button onClick={() => printDocument(`Spesenblatt_${event.date}`)} disabled={paid.length === 0}>
          Drucken
        </button>
      </div>
      <p>
        Event vom {formatDate(event.date)}, {event.startTime}–{event.endTime} Uhr
      </p>
      <EventInfo event={event} />
      {error && <p className="error">{error}</p>}

      {paid.length === 0 ? (
        <p className="muted">Niemand mit Betrag eingeteilt (Event bearbeiten).</p>
      ) : (
        <>
          {open > 0 && <p className="muted small">{open} von {paid.length} Empfängen noch nicht bestätigt.</p>}
          <table className="table payout">
            <thead>
              <tr>
                <th>Name</th>
                <th>Funktion</th>
                <th className="num">Betrag</th>
                <th>Erhalten</th>
              </tr>
            </thead>
            <tbody>
              {paid.map((e) => (
                <tr key={e.userId}>
                  <td>{e.name}</td>
                  <td>{e.functionName}</td>
                  <td className="num">{formatCHF(e.salary ?? 0)}</td>
                  <td className="signature-cell">
                    {e.signature ? (
                      <>
                        <img className="signature" src={`data:image/png;base64,${e.signature}`} alt={`Unterschrift ${e.name}`} />
                        <span className="muted small">
                          {e.signedAt && new Date(e.signedAt).toLocaleString('de-CH', { dateStyle: 'short', timeStyle: 'short' })}
                          <button className="link small" onClick={() => unsign(e)} title="Unterschrift entfernen">
                            ✕
                          </button>
                        </span>
                      </>
                    ) : (
                      <button className="small primary" onClick={() => setSigning(e)}>
                        Unterschreiben
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={2}>Total</td>
                <td className="num">{formatCHF(total)}</td>
                <td />
              </tr>
            </tfoot>
          </table>
        </>
      )}
      {byTime.length > 0 && (
        <p className="muted small">
          Über die Zeiterfassung abgerechnet: {byTime.map((e) => `${e.name} (${e.functionName})`).join(', ')}
        </p>
      )}

      <PayoutSheet event={event} paid={paid} byTime={byTime} />

      {signing && (
        <SignDialog
          eventId={event.id}
          entry={signing}
          onClose={() => setSigning(null)}
          onSigned={() => {
            setSigning(null)
            load()
          }}
        />
      )}
    </div>
  )
}

function SignDialog({
  eventId,
  entry,
  onClose,
  onSigned,
}: {
  eventId: number
  entry: PayoutEntry
  onClose: () => void
  onSigned: () => void
}) {
  const [signature, setSignature] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  async function confirmReceipt() {
    if (!signature) return
    setSaving(true)
    setError('')
    try {
      await api.put(`/events/${eventId}/payout/${entry.userId}`, { signature })
      onSigned()
    } catch (err) {
      setError(errorMessage(err))
      setSaving(false)
    }
  }

  return (
    <div className="modal" role="dialog" aria-modal="true" aria-label={`Unterschrift ${entry.name}`}>
      <div className="card stack modal-body">
        <strong>{entry.name}</strong>
        <p>
          Ich bestätige den Empfang von <strong>{formatCHF(entry.salary ?? 0)}</strong> ({entry.functionName}).
        </p>
        <SignaturePad onChange={setSignature} />
        {error && <p className="error">{error}</p>}
        <div className="actions">
          <button onClick={onClose}>Abbrechen</button>
          <button className="primary" onClick={confirmReceipt} disabled={!signature || saving}>
            Empfang bestätigen
          </button>
        </div>
      </div>
    </div>
  )
}

/** Leere Zeilen für handschriftliche Ergänzungen (z.B. Verpflegung) */
const extraRows = 3

/** Spesenblatt zum Drucken: Einsatz und Unterschrift pro Person, Total pro Funktion, Unterschrift Projektleitung. */
function PayoutSheet({ event, paid, byTime }: { event: Event; paid: PayoutEntry[]; byTime: PayoutEntry[] }) {
  const total = paid.reduce((sum, e) => sum + (e.salary ?? 0), 0)
  // Total pro Funktion in der Reihenfolge der Liste
  const byFunction = new Map<string, number>()
  for (const e of paid) byFunction.set(e.functionName, (byFunction.get(e.functionName) ?? 0) + (e.salary ?? 0))

  return (
    <PrintSheet title="Spesenblatt: Löhne und Entschädigungen">
      <dl className="sheet-fields">
        <dt>Midnightsports</dt>
        <dd>{event.location.name}</dd>
        <dt>Datum</dt>
        <dd>
          {formatShortDate(event.date)}, {event.startTime}–{event.endTime} Uhr
        </dd>
      </dl>

      <table className="sheet-table">
        <thead>
          <tr>
            <th>Vorname / Name</th>
            <th>Funktion</th>
            <th className="num">Einsatz</th>
            <th className="sheet-signature">Betrag erhalten / Unterschrift</th>
          </tr>
        </thead>
        <tbody>
          {paid.map((e) => (
            <tr key={e.userId}>
              <td>{e.name}</td>
              <td>{e.functionName}</td>
              <td className="num">{formatCHF(e.salary ?? 0)}</td>
              <td className="sheet-signature">
                {e.signature && <img src={`data:image/png;base64,${e.signature}`} alt="" />}
              </td>
            </tr>
          ))}
          {Array.from({ length: extraRows }, (_, i) => (
            <tr key={`extra-${i}`}>
              <td />
              <td />
              <td />
              <td />
            </tr>
          ))}
        </tbody>
      </table>

      <div className="sheet-bottom">
        <table className="sheet-table sheet-totals">
          <tbody>
            {[...byFunction].map(([name, sum], i) => (
              <tr key={name}>
                {i === 0 && <th rowSpan={byFunction.size + 1}>Total</th>}
                <td>{name}</td>
                <td className="num">{formatCHF(sum)}</td>
              </tr>
            ))}
            <tr className="sheet-grand-total">
              <td>Gesamt</td>
              <td className="num">{formatCHF(total)}</td>
            </tr>
          </tbody>
        </table>
        <div className="sheet-sign">
          <span>Unterschrift Projektleitung</span>
          <div className="sheet-sign-line" />
          <span className="sheet-small">{event.leader?.name}</span>
        </div>
      </div>

      {byTime.length > 0 && (
        <p className="sheet-small">
          Über die Zeiterfassung abgerechnet: {byTime.map((e) => `${e.name} (${e.functionName})`).join(', ')}
        </p>
      )}
    </PrintSheet>
  )
}
