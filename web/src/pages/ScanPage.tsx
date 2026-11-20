import QrScanner from 'qr-scanner'
import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { errorMessage, type CheckinResult } from '../api'
import { SanctionNotice } from '../components/Sanctions'
import { age, formatAddress, formatShortDate, genderLabels, relationLabels, statusLabels } from '../format'
import { useProject } from '../project'

/**
 * Scanner für Projektleiter und Mitarbeiter: QR-Code des Teilnehmers scannen und Person anzeigen.
 * Verwarnte Personen werden eingecheckt (gelbe Meldung), gesperrte nicht (rote Meldung).
 */
export default function ScanPage() {
  const { id } = useParams()
  const { api, base, project } = useProject()
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
  }, [api, id])

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
      <Link to={`${base}/events/${id}`} className="muted">
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
        <div className={`card stack scan-result ${scanState(result)}`}>
          <strong className="scan-status">
            {result.blocked
              ? '⛔ Gesperrt, nicht eingecheckt'
              : result.alreadyCheckedIn
                ? '⚠ Bereits eingecheckt'
                : '✓ Eingecheckt'}
          </strong>
          <SanctionNotice sanctions={result.sanctions} />
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
                <dd>{result.status ? statusLabels(project.region)[result.status] : '–'}</dd>
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

/** Rahmenfarbe: gesperrt rot, verwarnt oder bereits eingecheckt gelb, sonst grün */
function scanState(r: CheckinResult): string {
  if (r.blocked) return 'blocked'
  return r.alreadyCheckedIn || r.sanctions.length > 0 ? 'warn' : 'ok'
}
