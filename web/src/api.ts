export type Role = 'admin' | 'projektleiter' | 'teilnehmer'

export interface User {
  id: string
  email: string
  role: Role
  firstName: string
  lastName: string
  birthDate: string
  street: string
  zip: string
  city: string
  emergencyName: string
  emergencyPhone: string
  emergencyRelation: string
  profileComplete: boolean
}

export interface UserSummary {
  id: string
  email: string
  role: Role
  firstName: string
  lastName: string
  birthDate: string
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
  leaders: Person[]
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
  leaderIds: string[]
}

export interface Participant {
  checkinId: number | null
  userId: string | null
  name: string
  birthDate: string
  guest: boolean
  registered: boolean
  checkedInAt: string | null
}

export interface CheckinResult {
  checkinId: number
  person?: User
  guestName?: string
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
