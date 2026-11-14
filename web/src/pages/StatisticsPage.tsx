import { useEffect, useState } from 'react'
import { errorMessage, type Statistics } from '../api'
import PrintSheet, { printDocument } from '../components/PrintSheet'
import { attendanceLabel } from '../common/attendance'
import { lastMonths, perEvent, sum } from '../common/stats'
import { formatCHF, formatShortDate } from '../format'
import { useProject } from '../project'

/**
 * Statistik (Projektleiter und Admins) über vergangene, nicht abgesagte Events eines Zeitraums, Standard die letzten
 * 8 Monate: Teilnehmende pro Event nach Geschlecht, Einsätze des Teams, ausbezahlte Beträge (nur unterschriebene)
 * und Stunden der Zeiterfassung (PL, ohne festen Lohn).
 */
export default function StatisticsPage() {
  const { api } = useProject()
  const [range, setRange] = useState(() => lastMonths(8))
  const [stats, setStats] = useState<Statistics | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!range.from || !range.to) return
    // Antworten eines älteren Zeitraums ignorieren, wenn schnell nacheinander geändert wird
    let current = true
    setError('')
    api
      .get<Statistics>(`/statistics?from=${range.from}&to=${range.to}`)
      .then((s) => current && setStats(s))
      .catch((err) => current && setError(errorMessage(err)))
    return () => {
      current = false
    }
  }, [api, range])

  return (
    <div className="stack">
      <div className="page-header">
        <h1>Statistik</h1>
        <button onClick={() => printDocument(`Statistik_${range.from}_${range.to}`)} disabled={!stats}>
          Drucken
        </button>
      </div>
      <div className="row">
        <label>
          Von
          <input type="date" required value={range.from} onChange={(e) => setRange({ ...range, from: e.target.value })} />
        </label>
        <label>
          Bis
          <input type="date" required value={range.to} onChange={(e) => setRange({ ...range, to: e.target.value })} />
        </label>
      </div>
      {error && <p className="error">{error}</p>}
      {!stats && !error && <p className="muted">Laden…</p>}

      {stats && <StatisticsView stats={stats} />}
      {stats && <StatisticsSheet stats={stats} />}
    </div>
  )
}

/** Kennzahlen des Zeitraums, für Bildschirm und Druck gleich berechnet */
function summary(stats: Statistics) {
  const events = stats.eventCount
  return {
    events,
    average: perEvent(stats.attendance.total, events),
    byGender: attendanceLabel(stats.attendance, (n) => String(perEvent(n, events))),
    worked: sum(stats.team, (m) => m.events),
    paid: sum(stats.team, (m) => m.paid),
    hours: sum(stats.team, (m) => m.hours),
  }
}

function StatisticsView({ stats }: { stats: Statistics }) {
  const s = summary(stats)
  return (
    <>
      <p className="muted small">
        {formatShortDate(stats.from)} bis {formatShortDate(stats.to)}: vergangene Events ohne abgesagte.
      </p>
      <dl className="stat-grid wide" aria-label="Übersicht">
        <div className="stat">
          <dt>Events</dt>
          <dd>{s.events}</dd>
        </div>
        <div className="stat">
          <dt>Ø Teilnehmende</dt>
          <dd>{s.average}</dd>
        </div>
        <div className="stat">
          <dt>Ausbezahlt Mitarbeiter</dt>
          <dd>{formatCHF(s.paid)}</dd>
        </div>
        <div className="stat">
          <dt>Stunden PL</dt>
          <dd>{s.hours.toFixed(2)} h</dd>
        </div>
      </dl>
      <p className="small">
        Ø pro Event: {s.byGender}
        <span className="muted"> · total {stats.attendance.total} Check-ins</span>
      </p>

      <h2>Team</h2>
      {stats.team.length === 0 ? (
        <p className="muted">Im Zeitraum war niemand eingeteilt.</p>
      ) : (
        <div className="table-scroll">
          <TeamTable stats={stats} className="table" />
        </div>
      )}
    </>
  )
}

/** Einsätze, Beträge und Stunden pro Person mit Totalzeile, auf dem Bildschirm und im Druck */
function TeamTable({ stats, className }: { stats: Statistics; className: string }) {
  const s = summary(stats)
  return (
    <table className={className}>
      <thead>
        <tr>
          <th>Name</th>
          <th>Funktion</th>
          <th className="num">Einsätze</th>
          <th className="num">Ausbezahlt</th>
          <th className="num">Stunden</th>
        </tr>
      </thead>
      <tbody>
        {stats.team.map((m) => (
          <tr key={m.userId}>
            <td>{m.name}</td>
            <td>{m.function || '–'}</td>
            <td className="num">{m.events}</td>
            <td className="num">{m.paid > 0 ? formatCHF(m.paid) : '–'}</td>
            <td className="num">{m.hours > 0 ? m.hours.toFixed(2) : '–'}</td>
          </tr>
        ))}
      </tbody>
      <tfoot>
        <tr>
          <th colSpan={2}>Total</th>
          <th className="num">{s.worked}</th>
          <th className="num">{formatCHF(s.paid)}</th>
          <th className="num">{s.hours.toFixed(2)}</th>
        </tr>
      </tfoot>
    </table>
  )
}

/** Druckansicht (A4): Kennzahlen und Team des Zeitraums */
function StatisticsSheet({ stats }: { stats: Statistics }) {
  const s = summary(stats)
  return (
    <PrintSheet title="Statistik">
      <dl className="sheet-fields">
        <dt>Zeitraum</dt>
        <dd>
          {formatShortDate(stats.from)} bis {formatShortDate(stats.to)} (vergangene Events ohne abgesagte)
        </dd>
        <dt>Events</dt>
        <dd>{s.events}</dd>
        <dt>Ø Teilnehmende</dt>
        <dd>
          {s.average} pro Event ({s.byGender}), total {stats.attendance.total} Check-ins
        </dd>
        <dt>Ausbezahlt Mitarbeiter</dt>
        <dd>{formatCHF(s.paid)} (unterschrieben)</dd>
        <dt>Stunden PL</dt>
        <dd>{s.hours.toFixed(2)} h (Zeiterfassung)</dd>
      </dl>
      <h2 className="sheet-heading">Team</h2>
      {stats.team.length === 0 ? <p>Im Zeitraum war niemand eingeteilt.</p> : <TeamTable stats={stats} className="sheet-table" />}
    </PrintSheet>
  )
}
