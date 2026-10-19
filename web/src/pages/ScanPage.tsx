import QrScanner from 'qr-scanner'
import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, errorMessage, type CheckinResult } from '../api'
import { age, formatAddress, formatShortDate, genderLabels, relationLabels, statusLabels } from '../format'

/** Scanner für Projektleiter und Mitarbeiter: QR-Code des Teilnehmers scannen und Person anzeigen. */
export default function ScanPage() {
  const { id } = useParams()
  const video = useRef<HTMLVideoElement>(null)
  const scanner = useRef<QrScanner | null>(null)
  const busy = useRef(false)
  const [result, setResult] = useState<CheckinResult | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!video.current) return
    const s = new QrScanner(
      video.current,
      async (r) => {
        if (busy.current) return
        busy.current = true
        s.pause()
        try {
          setResult(await api.post<CheckinResult>(`/events/${id}/checkins`, { token: r.data }))
          setError('')
          navigator.vibrate?.(100)
        } catch (err) {
          setResult(null)
          setError(errorMessage(err))
        }
      },
      { preferredCamera: 'environment', highlightScanRegion: true, maxScansPerSecond: 5 },
    )
    scanner.current = s
    s.start().catch(() => setError('Kein Zugriff auf die Kamera. Bitte Berechtigung erteilen (HTTPS nötig).'))
    return () => s.destroy()
  }, [id])

  function next() {
    setResult(null)
    setError('')
    busy.current = false
    scanner.current?.start()
  }

  const p = result?.person
  const scanning = !result && !error
  return (
    <div className="narrow stack">
      <Link to={`/events/${id}`} className="muted">
        ← Zurück zum Event
      </Link>
      <h1>QR-Code scannen</h1>

      <div className={`scanner ${scanning ? '' : 'hidden'}`}>
        <video ref={video} muted playsInline />
      </div>

      {error && (
        <div className="card stack">
          <p className="error">{error}</p>
          <button className="primary" onClick={next}>
            Erneut scannen
          </button>
        </div>
      )}

      {result && (
        <div className={`card stack scan-result ${result.alreadyCheckedIn ? 'warn' : 'ok'}`}>
          <strong className="scan-status">
            {result.alreadyCheckedIn ? '⚠ Bereits eingecheckt' : '✓ Eingecheckt'}
          </strong>
          {p && (
            <>
              <h2>
                {p.firstName} {p.lastName}
              </h2>
              {!result.registered && <span className="badge">Nicht angemeldet</span>}
              <dl>
                <dt>Geburtsdatum</dt>
                <dd>
                  {formatShortDate(p.birthDate)} ({age(p.birthDate)} Jahre)
                </dd>
                <dt>Geschlecht</dt>
                <dd>{p.gender ? genderLabels[p.gender] : '–'}</dd>
                <dt>Status</dt>
                <dd>{p.status ? statusLabels[p.status] : '–'}</dd>
                <dt>Adresse</dt>
                <dd>{formatAddress(p)}</dd>
                <dt>Email</dt>
                <dd>{p.email}</dd>
                {p.emergencyPhone && <dt>Notfallkontakt</dt>}
                {p.emergencyPhone && (
                  <dd>
                  {p.emergencyName}
                  {p.emergencyRelation && ` (${relationLabels[p.emergencyRelation]})`}
                  <br />
                  <a href={`tel:${p.emergencyPhone}`}>{p.emergencyPhone}</a>
                </dd>
                )}
              </dl>
            </>
          )}
          <button className="primary" onClick={next}>
            Nächster Teilnehmer
          </button>
        </div>
      )}
    </div>
  )
}
