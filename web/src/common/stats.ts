import { todayISO } from '../format'

/** Summe eines Werts über eine Liste, z.B. Stunden der Zeiterfassung oder Beträge */
export function sum<T>(list: T[], value: (item: T) => number): number {
  return list.reduce((total, item) => total + value(item), 0)
}

/** Schnitt pro Event, auf eine Stelle gerundet. Ohne Events 0. */
export function perEvent(total: number, events: number): number {
  return events > 0 ? Math.round((total / events) * 10) / 10 : 0
}

/** Zeitraum der letzten Monate bis heute als ISO-Daten, Standard der Statistik: 8 Monate */
export function lastMonths(months: number, now: Date = new Date()): { from: string; to: string } {
  const from = new Date(now.getFullYear(), now.getMonth() - months, now.getDate())
  return { from: todayISO(from), to: todayISO(now) }
}
