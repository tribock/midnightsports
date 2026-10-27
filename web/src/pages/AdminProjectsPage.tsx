import { useCallback, useEffect, useState, type ChangeEvent, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api, errorMessage, type AdminProject } from '../api'
import { parseDecimal, readDataURL } from '../format'
import { projectLogo } from '../project'

/** Funktionen, wie sie Aadorf hat. Bezeichnung und Lohn sind beim Onboarding änderbar, leerer Lohn = Zeiterfassung. */
const defaultFunctions = [
  { name: 'JC', salary: '20' },
  { name: 'JSC', salary: '40' },
  { name: 'SC', salary: '80' },
  { name: 'PL', salary: '' },
]

/**
 * Globale Administration: alle Projekte mit Status und geteilten Verwarnungen. Neue Projekte werden hier erfasst
 * (Onboarding), mit Logo, Projekt-Admin, Funktionen, erstem Ort und optionalem Teilen mit bestehenden Projekten.
 */
export default function AdminProjectsPage() {
  const [projects, setProjects] = useState<AdminProject[] | null>(null)
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<AdminProject[]>('/admin/projects')
      .then(setProjects)
      .catch((err) => setError(errorMessage(err)))
  }, [])

  useEffect(load, [load])

  async function removeShare(p: AdminProject, partner: string) {
    if (!confirm(`Teilen zwischen ${p.name} und ${partner} entfernen?`)) return
    setError('')
    try {
      await api.del(`/admin/projects/${p.slug}/shares/${partner}`)
      load()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <div className="stack">
      <div className="page-header">
        <h1>Projekte</h1>
        {!creating && (
          <button className="primary" onClick={() => setCreating(true)}>
            Neues Projekt
          </button>
        )}
      </div>
      {error && <p className="error">{error}</p>}
      {creating && projects && (
        <ProjectForm
          projects={projects}
          onDone={() => {
            setCreating(false)
            load()
          }}
          onCancel={() => setCreating(false)}
        />
      )}
      {projects === null && !error && <p className="muted">Laden…</p>}
      <ul className="list">
        {projects?.map((p) =>
          editing === p.slug ? (
            <li key={p.slug} className="editing">
              <EditProjectForm
                project={p}
                onDone={() => {
                  setEditing(null)
                  load()
                }}
                onCancel={() => setEditing(null)}
              />
            </li>
          ) : (
            <li key={p.slug} className="project-row">
              <img src={projectLogo(p)} alt="" height={32} />
              <span className="grow">
                <strong>{p.name}</strong>
                {p.pending && <span className="badge warn">Wartet auf Einwilligung</span>}
                <br />
                <span className="muted small">
                  /{p.slug} · {p.region} · {p.memberCount} Mitglieder · Admins: {p.admins.join(', ') || '–'}
                </span>
                {p.shares.map((s) => (
                  <span key={s.partnerSlug} className="share-line small">
                    Teilt mit {s.partnerName}:{' '}
                    <span
                      className={s.state === 'eingewilligt' ? 'success' : s.state === 'abgelehnt' ? 'error' : 'muted'}
                    >
                      {s.state === 'ausstehend' && s.outgoing
                        ? `wartet auf Einwilligung von ${s.partnerName}`
                        : s.state}
                    </span>{' '}
                    <button className="link small" onClick={() => removeShare(p, s.partnerSlug)}>
                      entfernen
                    </button>
                  </span>
                ))}
              </span>
              <span className="actions">
                <button className="small" onClick={() => setEditing(p.slug)}>
                  Bearbeiten
                </button>
                <Link to={`/p/${p.slug}/events`} className="button small">
                  Öffnen
                </Link>
              </span>
            </li>
          ),
        )}
      </ul>
    </div>
  )
}

/** Logo wählen: PNG, JPEG oder WebP bis 512 KB (SVG ist nicht erlaubt). */
function LogoInput({ onChange }: { onChange: (dataURL: string | null) => void }) {
  const [error, setError] = useState('')

  async function pick(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    setError('')
    if (!file) return onChange(null)
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type))
      return setError('Bitte ein PNG-, JPEG- oder WebP-Bild wählen')
    if (file.size > 512 * 1024) return setError('Das Logo ist zu gross (höchstens 512 KB)')
    onChange(await readDataURL(file))
  }

  return (
    <label>
      Logo (PNG, JPEG oder WebP, leer = Standardlogo)
      <input type="file" accept="image/png,image/jpeg,image/webp" onChange={pick} />
      {error && <span className="error small">{error}</span>}
    </label>
  )
}

/** Onboarding eines neuen Projekts. Danach ist alles angelegt wie bei Aadorf. */
function ProjectForm({
  projects,
  onDone,
  onCancel,
}: {
  projects: AdminProject[]
  onDone: () => void
  onCancel: () => void
}) {
  const [form, setForm] = useState({ name: '', slug: '', region: '', description: '', adminEmail: '' })
  const [logo, setLogo] = useState<string | null>(null)
  const [functions, setFunctions] = useState(defaultFunctions)
  const [location, setLocation] = useState({ name: '', street: '', zip: '', city: '' })
  const [shareWith, setShareWith] = useState<string[]>([])
  const [error, setError] = useState('')

  // Kurzname aus dem Namen vorschlagen, solange er nicht selbst geändert wurde
  const [slugTouched, setSlugTouched] = useState(false)
  function setName(name: string) {
    setForm((f) => ({ ...f, name, slug: slugTouched ? f.slug : slugify(name) }))
  }

  function setFunction(i: number, field: 'name' | 'salary', value: string) {
    setFunctions((list) => list.map((f, j) => (i === j ? { ...f, [field]: value } : f)))
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (shareWith.length > 0) {
      const names = projects
        .filter((p) => shareWith.includes(p.slug))
        .map((p) => p.name)
        .join(', ')
      if (!confirm(`Das Projekt bleibt gesperrt, bis der Projekt-Admin von ${names} eingewilligt hat. Fortfahren?`))
        return
    }
    try {
      await api.post('/admin/projects', {
        ...form,
        logo,
        functions: functions
          .filter((f) => f.name.trim())
          .map((f) => ({ name: f.name, salary: parseDecimal(f.salary) })),
        location,
        shareWith,
      })
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <form className="card stack" onSubmit={submit}>
      <h2>Neues Projekt</h2>
      <div className="row">
        <label>
          Name
          <input
            required
            autoFocus
            value={form.name}
            onChange={(e) => setName(e.target.value)}
            placeholder="z.B. Pleno Frauenfeld"
          />
        </label>
        <label>
          Kurzname (für den Link)
          <input
            required
            pattern="[a-z0-9]+(-[a-z0-9]+)*"
            value={form.slug}
            onChange={(e) => {
              setSlugTouched(true)
              setForm({ ...form, slug: e.target.value })
            }}
            placeholder="pleno-frauenfeld"
          />
        </label>
      </div>
      <div className="row">
        <label>
          Region (für «Schüler-/in aus …»)
          <input
            required
            value={form.region}
            onChange={(e) => setForm({ ...form, region: e.target.value })}
            placeholder="Frauenfeld"
          />
        </label>
        <label>
          Email des Projekt-Admins
          <input
            required
            type="email"
            value={form.adminEmail}
            onChange={(e) => setForm({ ...form, adminEmail: e.target.value })}
          />
        </label>
      </div>
      <label>
        Beschreibung (für die Suche, freiwillig)
        <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
      </label>
      <LogoInput onChange={setLogo} />

      <fieldset className="stack">
        <legend>Funktionen</legend>
        <p className="muted small">
          Bezeichnung und Lohn pro Einsatz. Ohne Lohn wird über die Zeiterfassung abgerechnet.
        </p>
        {functions.map((f, i) => (
          <div key={i} className="row">
            <label>
              Bezeichnung
              <input value={f.name} onChange={(e) => setFunction(i, 'name', e.target.value)} />
            </label>
            <label>
              Lohn (CHF)
              <input
                inputMode="decimal"
                pattern="[0-9]+([.,][0-9]{1,2})?"
                value={f.salary}
                onChange={(e) => setFunction(i, 'salary', e.target.value)}
                placeholder="leer = Zeiterfassung"
              />
            </label>
          </div>
        ))}
        <div className="actions">
          <button
            type="button"
            className="small"
            onClick={() => setFunctions([...functions, { name: '', salary: '' }])}
          >
            + Funktion
          </button>
        </div>
      </fieldset>

      <fieldset className="stack">
        <legend>Erster Veranstaltungsort</legend>
        <div className="row">
          <label>
            Name
            <input
              required
              value={location.name}
              onChange={(e) => setLocation({ ...location, name: e.target.value })}
            />
          </label>
          <label>
            Strasse und Nr.
            <input value={location.street} onChange={(e) => setLocation({ ...location, street: e.target.value })} />
          </label>
        </div>
        <div className="row">
          <label className="zip">
            PLZ
            <input value={location.zip} onChange={(e) => setLocation({ ...location, zip: e.target.value })} />
          </label>
          <label>
            Ort
            <input value={location.city} onChange={(e) => setLocation({ ...location, city: e.target.value })} />
          </label>
        </div>
      </fieldset>

      {projects.length > 0 && (
        <fieldset className="stack">
          <legend>Verwarnungen und Sperren teilen mit</legend>
          <p className="muted small">
            Beide Projekte sehen dann die Verwarnungen und Sperren des anderen. Das neue Projekt bleibt gesperrt, bis
            der Projekt-Admin des Partners eingewilligt hat.
          </p>
          {projects.map((p) => (
            <label key={p.slug} className="check">
              <input
                type="checkbox"
                checked={shareWith.includes(p.slug)}
                onChange={() =>
                  setShareWith((list) => (list.includes(p.slug) ? list.filter((s) => s !== p.slug) : [...list, p.slug]))
                }
              />
              {p.name}
            </label>
          ))}
        </fieldset>
      )}

      {error && <p className="error">{error}</p>}
      <div className="actions">
        <button type="button" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="primary">
          Projekt erstellen
        </button>
      </div>
    </form>
  )
}

/** Name, Region, Beschreibung und Logo eines bestehenden Projekts ändern. */
function EditProjectForm({
  project,
  onDone,
  onCancel,
}: {
  project: AdminProject
  onDone: () => void
  onCancel: () => void
}) {
  const [form, setForm] = useState({ name: project.name, region: project.region, description: project.description })
  // undefined = Logo unverändert, "" = Standardlogo, sonst das neue Logo
  const [logo, setLogo] = useState<string | undefined>(undefined)
  const [error, setError] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.put(`/admin/projects/${project.slug}`, { ...form, logo: logo ?? null })
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
          <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        </label>
        <label>
          Region
          <input required value={form.region} onChange={(e) => setForm({ ...form, region: e.target.value })} />
        </label>
      </div>
      <label>
        Beschreibung
        <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
      </label>
      <LogoInput onChange={(v) => setLogo(v ?? undefined)} />
      {project.hasLogo && (
        <label className="check">
          <input type="checkbox" checked={logo === ''} onChange={(e) => setLogo(e.target.checked ? '' : undefined)} />
          Standardlogo verwenden
        </label>
      )}
      {error && <p className="error">{error}</p>}
      <div className="actions">
        <button type="button" className="small" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="small primary">
          Speichern
        </button>
      </div>
    </form>
  )
}

/** "Pleno Frauenfeld" → "pleno-frauenfeld" (Umlaute ausgeschrieben) */
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/ä/g, 'ae')
    .replace(/ö/g, 'oe')
    .replace(/ü/g, 'ue')
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
}
