import type { Attendance } from '../api'

/** "W 12 · M 10 · D 1 · ohne Angabe 2" (ohne Angabe nur, wenn es solche gibt). Bericht und Statistik. */
export function attendanceLabel(a: Attendance, format: (n: number) => string = String): string {
  const parts = [`W ${format(a.w)}`, `M ${format(a.m)}`, `D ${format(a.d)}`]
  if (a.other > 0) parts.push(`ohne Angabe ${format(a.other)}`)
  return parts.join(' · ')
}
