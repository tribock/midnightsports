import type { Role } from './api'

export const roleLabels: Record<Role, string> = {
  admin: 'Admin',
  projektleiter: 'Projektleiter',
  teilnehmer: 'Teilnehmer',
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
