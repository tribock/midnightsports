export type Role = 'admin' | 'projektleiter' | 'mitarbeiter' | 'teilnehmer'
export type Gender = 'w' | 'm' | 'd'
export type EmergencyRelation = 'vater' | 'mutter' | 'andere'
export type UserStatus = 'schueler_aadorf' | 'gast_extern' | 'mitarbeiter'

export interface User {
  id: string
  email: string
  role: Role
  firstName: string
  lastName: string
  birthDate: string
  gender: Gender | ''
  status: UserStatus | ''
  street: string
  zip: string
  city: string
  emergencyName: string
  emergencyPhone: string
  emergencyRelation: EmergencyRelation | ''
  /** ohne Login erfasst, Email freiwillig */
  guest: boolean
  /** darf Notfallkontakte anderer Benutzer einsehen (Rolle oder Funktion mit Zugriff, siehe Administration) */
  emergencyAccess: boolean
  profileComplete: boolean
}

export interface UserSummary {
  id: string
  email: string
  role: Role
  firstName: string
  lastName: string
  birthDate: string
  guest: boolean
}

export interface Location {
  id: number
  name: string
  street: string
  zip: string
  city: string
  isDefault: boolean
}

export interface Person {
  id: string
  name: string
}

export interface Event {
  id: number
  date: string
  startTime: string
  endTime: string
  startsAt: string
  endsAt: string
  cancelled: boolean
  location: Location
  leader: Person | null
  registered: boolean
  registrationCount: number
  checkinCount: number
  registrationOpen: boolean
}

export interface EventInput {
  date: string
  startTime: string
  endTime: string
  locationId: number
  leaderId: string
  /** Einsatz, nur Projektleiter und Admins */
  staff?: { userId: string }[]
}

export interface Participant {
  checkinId: number | null
  userId: string | null
  name: string
  birthDate: string
  status: UserStatus | ''
  guest: boolean
  registered: boolean
  checkedInAt: string | null
}

export interface CheckinResult {
  checkinId: number
  person: User
  registered: boolean
  alreadyCheckedIn: boolean
  checkedInAt: string
}

export interface Report {
  eventId: number
  body: string
  authorName: string
  updatedAt: string | null
  registrationCount: number
  checkinCount: number
  /** Anwesende nach Geschlecht, other = ohne Angabe (z.B. gelöschte Konten) */
  attendance: { w: number; m: number; d: number; other: number }
}

export interface TimeEntry {
  id: number
  userId: string
  userName: string
  date: string
  hours: number
  description: string
  eventId: number | null
  eventDate: string
}

/** Funktion im Team, salary null = Abrechnung über die Zeiterfassung (z.B. PL) */
export interface Func {
  id: number
  name: string
  salary: number | null
}

/** Rollen und Funktionen, die Notfallkontakte einsehen dürfen (Administration). Admins immer. */
export interface EmergencyAccess {
  roles: Role[]
  functionIds: number[]
}

export interface EmergencyContact {
  name: string
  phone: string
  relation: EmergencyRelation | ''
}

export interface TeamMember {
  id: string
  email: string
  role: Role
  name: string
  functionId: number | null
  functionName: string | null
}

/** Einsatz einer Person an einem Event, salary nur für Projektleiter und Admins */
export interface StaffEntry {
  userId: string
  name: string
  functionId: number
  functionName: string
  salary?: number
  /** Abrechnung über die Zeiterfassung statt Betrag am Event */
  paidByTime: boolean
}

/** Zeile der Auszahlung an einem Event, salary null = Abrechnung über die Zeiterfassung */
export interface PayoutEntry {
  userId: string
  name: string
  functionName: string
  salary: number | null
  paidByTime: boolean
  /** Unterschrift als PNG, base64-kodiert */
  signature?: string
  signedAt: string | null
  signedByName: string | null
}

/** Auszahlung an einem Event, abgeschlossen mit der Unterschrift der Projektleitung */
export interface Payout {
  entries: PayoutEntry[]
  /** Unterschrift der Projektleitung als PNG, base64-kodiert */
  signature?: string
  closedAt: string | null
  closedByName: string | null
}

export interface BuildInfo {
  version: string
  date: string
}

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch('/api' + path, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: 'same-origin',
  })
  if (res.status === 204) {
    return undefined as T
  }
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError(data.error ?? 'Unbekannter Fehler', res.status)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body: unknown = {}) => request<T>('POST', path, body),
  put: <T>(path: string, body: unknown) => request<T>('PUT', path, body),
  del: <T = void>(path: string) => request<T>('DELETE', path),
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'Unbekannter Fehler'
}
