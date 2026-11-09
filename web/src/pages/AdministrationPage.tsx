import { useCallback, useEffect, useState, type FormEvent } from 'react'
import QRCode from 'react-qr-code'
import {
  errorMessage,
  type BanAction,
  type EmergencyAccess,
  type Func,
  type ProjectShare,
  type Role,
  type SanctionRules,
} from '../api'
import AdminLocations from '../components/AdminLocations'
import { banActionLabels, parseDecimal, roleLabels, salaryLabel } from '../format'
import { useProject } from '../project'

/**
 * Administration des Projekts (Projekt-Admins): Projekt-Link, Funktionen mit Lohn pro Einsatz, Zugriff auf
 * Notfallkontakte, Fristen für Verwarnungen und Sperren, geteilte Verwarnungen und Veranstaltungsorte.
 */
export default function AdministrationPage() {
  const { api, project } = useProject()
  const [functions, setFunctions] = useState<Func[]>([])
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<Func[]>('/functions')
      .then(setFunctions)
      .catch((err) => setError(errorMessage(err)))
  }, [api])

  useEffect(load, [load])

  return (
    <div className="stack">
      <h1>Administration</h1>
      <p className="muted">{project.name}</p>
      {error && <p className="error">{error}</p>}
      <JoinLink />
      <Functions functions={functions} onChanged={load} />
      <EmergencyAccessSettings functions={functions} />
      <SanctionSettings />
      <Shares />
      <AdminLocations />
    </div>
  )
}

/**
 * Link und QR-Code zum Anmelden beim Projekt, z.B. für ein Plakat in der Turnhalle. Wer den Code scannt,
 * landet nach dem Login direkt auf der Anmeldung für dieses Projekt.
 */
function JoinLink() {
  const { project } = useProject()
  const url = `${window.location.origin}/?projekt=${project.slug}`
  return (
    <section className="stack">
      <div className="section-header">
        <h2>Projekt-Link</h2>
      </div>
      <p className="muted small">
        Über diesen Link oder QR-Code melden sich Teilnehmende für {project.name} an (z.B. als Plakat aushängen).
      </p>
      <div className="card join-link">
        <div className="qr">
          <QRCode value={url} size={160} />
        </div>
        <a href={url}>{url}</a>
      </div>
    </section>
  )
}

/** Funktionen als Liste, Bearbeiten direkt in der Zeile. */
function Functions({ functions, onChanged }: { functions: Func[]; onChanged: () => void }) {
  // id der bearbeiteten Funktion, 0 = neue Funktion
  const [editing, setEditing] = useState<number | null>(null)

  function done() {
    setEditing(null)
    onChanged()
  }

  return (
    <section className="stack">
      <div className="section-header">
        <h2>Funktionen</h2>
        {editing !== 0 && (
          <button className="small" onClick={() => setEditing(0)}>
            + Neue Funktion
          </button>
        )}
      </div>
      <p className="muted small">
        Lohn pro Einsatz an einem Event. Ohne Lohn (z.B. PL) wird über die Zeiterfassung abgerechnet.
      </p>
      <ul className="list">
        {functions.map((f) =>
          editing === f.id ? (
            <li key={f.id} className="editing">
              <FunctionForm fn={f} onDone={done} onCancel={() => setEditing(null)} />
            </li>
          ) : (
            <li key={f.id}>
              <strong className="grow">{f.name}</strong>
              <span className={f.salary === null ? 'muted small' : 'num'}>{salaryLabel(f.salary)}</span>
              <button className="small" onClick={() => setEditing(f.id)}>
                Bearbeiten
              </button>
            </li>
          ),
        )}
        {editing === 0 && (
          <li className="editing">
            <FunctionForm fn={{ id: 0, name: '', salary: null }} onDone={done} onCancel={() => setEditing(null)} />
          </li>
        )}
        {functions.length === 0 && editing !== 0 && <li className="muted">Noch keine Funktionen erfasst.</li>}
      </ul>
    </section>
  )
}

function FunctionForm({ fn, onDone, onCancel }: { fn: Func; onDone: () => void; onCancel: () => void }) {
  const { api } = useProject()
  const [name, setName] = useState(fn.name)
  const [salary, setSalary] = useState(fn.salary === null ? '' : String(fn.salary))
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    const body = { name, salary: parseDecimal(salary) }
    try {
      if (fn.id) {
        await api.put(`/functions/${fn.id}`, body)
      } else {
        await api.post('/functions', body)
      }
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function remove() {
    if (!confirm(`Funktion ${fn.name} löschen?`)) return
    try {
      await api.del(`/functions/${fn.id}`)
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="stack" onSubmit={submit}>
      <div className="row">
        <label>
          Name
          <input required autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder="z.B. JC" />
        </label>
        <label>
          Lohn pro Einsatz (CHF)
          <input
            inputMode="decimal"
            pattern="[0-9]+([.,][0-9]{1,2})?"
            value={salary}
            onChange={(e) => setSalary(e.target.value)}
            placeholder="leer = über Zeiterfassung"
          />
        </label>
      </div>
      {error && <p className="error">{error}</p>}
      <div className="actions">
        {fn.id !== 0 && (
          <button type="button" className="small danger" onClick={remove}>
            Löschen
          </button>
        )}
        <span className="grow" />
        <button type="button" className="small" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="small primary">
          {fn.id ? 'Speichern' : 'Hinzufügen'}
        </button>
      </div>
    </form>
  )
}

/** Wählbare Rollen: Admins haben immer Zugriff auf Notfallkontakte, Teilnehmer nie */
const accessRoles: Role[] = ['projektleiter', 'mitarbeiter']

/** Rollen und Funktionen, die Notfallkontakte einsehen dürfen. Jede Änderung wird sofort gespeichert. */
function EmergencyAccessSettings({ functions }: { functions: Func[] }) {
  const { api, reload } = useProject()
  const [access, setAccess] = useState<EmergencyAccess | null>(null)
  const [error, setError] = useState('')

  // Neu laden, wenn Funktionen ändern (gelöschte Funktionen verlieren ihren Zugriff)
  useEffect(() => {
    api
      .get<EmergencyAccess>('/emergency-access')
      .then(setAccess)
      .catch((err) => setError(errorMessage(err)))
  }, [api, functions])

  function toggle<T>(list: T[], value: T): T[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
  }

  async function save(next: EmergencyAccess) {
    const previous = access
    setAccess(next)
    setError('')
    try {
      setAccess(await api.put<EmergencyAccess>('/emergency-access', next))
      // Der eigene Zugriff kann sich geändert haben
      await reload()
    } catch (err) {
      setAccess(previous)
      setError(errorMessage(err))
    }
  }

  if (!access) return error ? <p className="error">{error}</p> : null
  return (
    <section className="stack">
      <div className="section-header">
        <h2>Zugriff auf Notfallkontakte</h2>
      </div>
      <p className="muted small">
        Gewählte Rollen und Funktionen dürfen die Notfallkontakte aller Mitglieder einsehen. Projekt-Admins immer,
        Teilnehmer nie. Jeder Abruf wird im Audit-Protokoll festgehalten.
      </p>
      <div className="card access-grid">
        <span className="muted small">Rollen</span>
        <div className="chips">
          <button className="chip active" disabled title="Admins haben immer Zugriff">
            {roleLabels.admin}
          </button>
          {accessRoles.map((role) => (
            <button
              key={role}
              className={access.roles.includes(role) ? 'chip active' : 'chip'}
              aria-pressed={access.roles.includes(role)}
              onClick={() => save({ ...access, roles: toggle(access.roles, role) })}
            >
              {roleLabels[role]}
            </button>
          ))}
        </div>
        <span className="muted small">Funktionen</span>
        <div className="chips">
          {functions.length === 0 && <span className="muted small">Noch keine Funktionen erfasst.</span>}
          {functions.map((f) => (
            <button
              key={f.id}
              className={access.functionIds.includes(f.id) ? 'chip active' : 'chip'}
              aria-pressed={access.functionIds.includes(f.id)}
              onClick={() => save({ ...access, functionIds: toggle(access.functionIds, f.id) })}
            >
              {f.name}
            </button>
          ))}
        </div>
      </div>
      {error && <p className="error">{error}</p>}
    </section>
  )
}

/** Tage als Eingabe: leer = ohne Frist */
function daysValue(n: number | null): string {
  return n === null ? '' : String(n)
}

function parseDays(s: string): number | null {
  return s.trim() === '' ? null : Number(s)
}

/**
 * Fristen für Verwarnungen und Sperren: nach Ablauf wird eine Sperre aufgehoben oder zur Verwarnung,
 * eine Verwarnung gelöscht. Gilt auch für bestehende Einträge.
 */
function SanctionSettings() {
  const { api, project, reload } = useProject()
  const [banDays, setBanDays] = useState(daysValue(project.banDays))
  const [banAction, setBanAction] = useState<BanAction>(project.banAction)
  const [warningDays, setWarningDays] = useState(daysValue(project.warningDays))
  const [status, setStatus] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setStatus('')
    try {
      const rules: SanctionRules = { banDays: parseDays(banDays), banAction, warningDays: parseDays(warningDays) }
      await api.put('/settings/sanctions', rules)
      await reload()
      setStatus('Gespeichert')
    } catch (err) {
      setStatus(errorMessage(err))
    }
  }

  return (
    <section className="stack">
      <div className="section-header">
        <h2>Verwarnungen und Sperren</h2>
      </div>
      <p className="muted small">
        Fristen ab dem Erfassen. Leer = gilt, bis sie jemand aufhebt. Änderungen gelten auch für bestehende Einträge.
      </p>
      <form className="card stack" onSubmit={submit}>
        <div className="row">
          <label>
            Sperre gilt (Tage)
            <input
              inputMode="numeric"
              pattern="[0-9]*"
              value={banDays}
              onChange={(e) => setBanDays(e.target.value)}
              placeholder="unbefristet"
            />
          </label>
          <label>
            Danach
            <select
              value={banAction}
              onChange={(e) => setBanAction(e.target.value as BanAction)}
              disabled={banDays === ''}
            >
              {Object.entries(banActionLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label>
          Verwarnung gilt (Tage), danach wird sie gelöscht
          <input
            inputMode="numeric"
            pattern="[0-9]*"
            value={warningDays}
            onChange={(e) => setWarningDays(e.target.value)}
            placeholder="unbefristet"
          />
        </label>
        <div className="actions">
          {status && <span className="muted small">{status}</span>}
          <button type="submit" className="primary">
            Speichern
          </button>
        </div>
      </form>
    </section>
  )
}

const shareStateLabels: Record<ProjectShare['state'], string> = {
  ausstehend: 'Wartet auf Einwilligung',
  eingewilligt: 'Aktiv',
  abgelehnt: 'Abgelehnt',
}

/**
 * Geteilte Verwarnungen und Sperren: Anfragen anderer Projekte annehmen oder ablehnen, aktives Teilen beenden.
 * Ein anfragendes Projekt bleibt gesperrt, bis hier eingewilligt wurde.
 */
function Shares() {
  const { api } = useProject()
  const [shares, setShares] = useState<ProjectShare[] | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<ProjectShare[]>('/shares')
      .then(setShares)
      .catch((err) => setError(errorMessage(err)))
  }, [api])

  useEffect(load, [load])

  async function act(s: ProjectShare, action: 'consent' | 'decline' | 'end') {
    const question = {
      consent: `Verwarnungen und Sperren mit ${s.partnerName} teilen? Beide Projekte sehen dann die Einträge des anderen.`,
      decline: `Anfrage von ${s.partnerName} ablehnen?`,
      end: `Teilen mit ${s.partnerName} beenden?`,
    }[action]
    if (!confirm(question)) return
    setError('')
    try {
      if (action === 'end') await api.del(`/shares/${s.partnerSlug}`)
      else await api.post(`/shares/${s.partnerSlug}/${action}`)
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (!shares) return error ? <p className="error">{error}</p> : null
  return (
    <section className="stack">
      <div className="section-header">
        <h2>Geteilte Verwarnungen</h2>
      </div>
      <p className="muted small">
        Mit Partnerprojekten geteilte Verwarnungen und Sperren erscheinen auch dort, z.B. beim Einchecken. Ändern kann
        sie nur das Projekt, das sie erfasst hat. Anfragen stellt die globale Administration beim Erstellen eines
        Projekts.
      </p>
      {error && <p className="error">{error}</p>}
      <ul className="list">
        {shares.length === 0 && <li className="muted">Kein Teilen eingerichtet.</li>}
        {shares.map((s) => (
          <li key={s.partnerSlug}>
            <span className="grow">
              <strong>{s.partnerName}</strong>
              <span
                className={`badge ${s.state === 'eingewilligt' ? 'success' : s.state === 'abgelehnt' ? 'danger' : 'warn'}`}
              >
                {shareStateLabels[s.state]}
              </span>
              <br />
              <span className="muted small">{s.outgoing ? 'Anfrage dieses Projekts' : 'Anfrage des Partners'}</span>
            </span>
            {!s.outgoing && s.state !== 'eingewilligt' && (
              <>
                {s.state === 'ausstehend' && (
                  <button className="small" onClick={() => act(s, 'decline')}>
                    Ablehnen
                  </button>
                )}
                <button className="small primary" onClick={() => act(s, 'consent')}>
                  Einwilligen
                </button>
              </>
            )}
            {s.state === 'eingewilligt' && (
              <button className="small danger" onClick={() => act(s, 'end')}>
                Beenden
              </button>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}
