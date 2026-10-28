import type { BanAction, EmergencyRelation, Gender, Role, SanctionLevel, UserStatus } from './api'

export const roleLabels: Record<Role, string> = {
  admin: 'Projekt-Admin',
  projektleiter: 'Projektleiter',
  mitarbeiter: 'Mitarbeiter',
  teilnehmer: 'Teilnehmer',
}

export const genderLabels: Record<Gender, string> = {
  w: 'Weiblich',
  m: 'Männlich',
  d: 'Divers',
}

export const relationLabels: Record<EmergencyRelation, string> = {
  vater: 'Vater',
  mutter: 'Mutter',
  andere: 'Andere',
}

/** Status im Projekt, "Schüler/in" mit der Region des Projekts: statusLabels("Aadorf").schueler */
export function statusLabels(region: string): Record<UserStatus, string> {
  return {
    schueler: region ? `Schüler-/in aus ${region}` : 'Schüler-/in',
    gast_extern: 'Gast extern',
    mitarbeiter: 'Mitarbeiter',
  }
}

export const sanctionLabels: Record<SanctionLevel, string> = {
  verwarnung: 'Verwarnt',
  sperre: 'Gesperrt',
}

export const banActionLabels: Record<BanAction, string> = {
  entsperren: 'Sperre aufheben',
  verwarnen: 'In Verwarnung umwandeln',
}

/** Aktionen im Audit-Protokoll (siehe audit.go) */
export const auditLabels: Record<string, string> = {
  'mitglied.angemeldet': 'Beim Projekt angemeldet',
  'mitglied.rolle': 'Rolle geändert',
  'mitglied.status': 'Status geändert',
  'mitglied.funktion': 'Funktion geändert',
  'gast.erfasst': 'Gast erfasst',
  'notfallkontakt.angesehen': 'Notfallkontakt angesehen',
  'profil.angesehen': 'Profil angesehen',
  'profil.geaendert': 'Profil geändert',
  'benutzer.geloescht': 'Benutzer gelöscht',
  'konto.geloescht': 'Eigenes Konto gelöscht',
  export: 'Daten exportiert',
  'sanktion.erfasst': 'Verwarnt / gesperrt',
  'sanktion.aufgehoben': 'Verwarnung / Sperre aufgehoben',
  'sanktion.fristen': 'Fristen für Verwarnungen geändert',
  'notfallzugriff.geaendert': 'Zugriff auf Notfallkontakte geändert',
  'projekt.erstellt': 'Projekt erstellt',
  'projekt.geaendert': 'Projekt geändert',
  'teilen.angefragt': 'Teilen angefragt',
  'teilen.eingewilligt': 'Teilen eingewilligt',
  'teilen.abgelehnt': 'Teilen abgelehnt',
  'teilen.entfernt': 'Teilen beendet',
}

/** "Max Muster", ohne Namen die Email bzw. "(Profil unvollständig)" */
export function personName(p: { firstName: string; lastName: string; email?: string }): string {
  const name = `${p.firstName} ${p.lastName}`.trim()
  return name || p.email || '(Profil unvollständig)'
}

/** Datei als data-URL lesen (Logo-Upload) */
export function readDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

const chf = new Intl.NumberFormat('de-CH', { style: 'currency', currency: 'CHF' })

/** Lohn einer Funktion: Betrag oder Abrechnung über die Zeiterfassung */
export function salaryLabel(salary: number | null): string {
  return salary === null ? 'über Zeiterfassung' : formatCHF(salary)
}

/** 40 → "CHF 40.00" */
export function formatCHF(amount: number): string {
  return chf.format(amount)
}

/** Eingabe mit Komma oder Punkt: "3,5" → 3.5. Leere Eingabe → null */
export function parseDecimal(input: string): number | null {
  const s = input.trim()
  return s === '' ? null : Number(s.replace(',', '.'))
}

/** "Hauptstrasse 1, 8355 Aadorf", fehlende Teile werden weggelassen */
export function formatAddress(a: { street: string; zip: string; city: string }): string {
  return [a.street, `${a.zip} ${a.city}`.trim()].filter(Boolean).join(', ')
}

function parseDate(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d)
}

/** "2026-11-07" → "Sa., 07.11.2026" */
export function formatDate(iso: string): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat('de-CH', {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(parseDate(iso))
}

/** "2026-11-07" → "07.11.2026" */
export function formatShortDate(iso: string): string {
  if (!iso) return ''
  const [y, m, d] = iso.split('-')
  return `${d}.${m}.${y}`
}

export function formatTime(isoDateTime: string): string {
  return new Intl.DateTimeFormat('de-CH', { hour: '2-digit', minute: '2-digit' }).format(new Date(isoDateTime))
}

/** Alter in Jahren am Stichtag */
export function age(birthDate: string, on: Date = new Date()): number | null {
  if (!birthDate) return null
  const b = parseDate(birthDate)
  let years = on.getFullYear() - b.getFullYear()
  if (on.getMonth() < b.getMonth() || (on.getMonth() === b.getMonth() && on.getDate() < b.getDate())) {
    years--
  }
  return years
}

export function todayISO(now: Date = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}
