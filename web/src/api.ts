/** Rolle in einem Projekt. admin = Projekt-Admin */
export type Role = 'admin' | 'projektleiter' | 'mitarbeiter' | 'teilnehmer'
export type Gender = 'w' | 'm' | 'd'
export type EmergencyRelation = 'vater' | 'mutter' | 'andere'
/** Herkunft im Projekt, gewählt beim Anmelden. schueler = Schüler/in aus der Region des Projekts */
export type UserStatus = 'schueler' | 'gast_extern' | 'mitarbeiter'
export type SanctionLevel = 'verwarnung' | 'sperre'
export type BanAction = 'entsperren' | 'verwarnen'

/** Person, unabhängig vom Projekt. Rolle und Status gehören zur Mitgliedschaft (Member). */
export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  birthDate: string
  gender: Gender | ''
  street: string
  zip: string
  city: string
  emergencyName: string
  emergencyPhone: string
  emergencyRelation: EmergencyRelation | ''
  /** ohne Login erfasst, Email freiwillig */
  guest: boolean
  /** verwaltet Projekte, Benutzer und das Audit-Protokoll, ist in jedem Projekt Admin */
  globalAdmin: boolean
  profileComplete: boolean
}

/** Projekt, bei dem der angemeldete Benutzer Mitglied ist (Projektauswahl) */
export interface MemberProject {
  slug: string
  name: string
  role: Role
  /** wartet auf die Einwilligung eines Partnerprojekts */
  pending: boolean
}

/** Angemeldeter Benutzer mit seinen Projekten (GET /me) */
export interface Me extends User {
  projects: MemberProject[]
}

export interface SanctionRules {
  /** nach so vielen Tagen wird eine Sperre aufgehoben bzw. zur Verwarnung (banAction), null = ohne Frist */
  banDays: number | null
  banAction: BanAction
  /** nach so vielen Tagen wird eine Verwarnung gelöscht, null = ohne Frist */
  warningDays: number | null
}

export interface Project extends SanctionRules {
  slug: string
  name: string
  region: string
  description: string
  hasLogo: boolean
  pending: boolean
}

/** Angemeldeter Benutzer im Kontext eines Projekts (GET /p/{slug}/me) */
export interface Member extends User {
  project: Project
  role: Role
  status: UserStatus | ''
  functionId: number | null
  /** darf Notfallkontakte der Mitglieder einsehen (Rolle oder Funktion mit Zugriff, siehe Administration) */
  emergencyAccess: boolean
  /** echte Mitgliedschaft (globale Admins ohne Mitgliedschaft: false) */
  isMember: boolean
}

/** Geltende Verwarnung oder Sperre, wie das aktuelle Projekt sie sieht */
export interface Sanction {
  level: SanctionLevel
  reason: string
  since: string
  /** Ende der aktuellen Stufe, null = unbefristet */
  until: string | null
  projectName: string
  /** im eigenen Projekt erfasst (änderbar), sonst von einem Partnerprojekt geteilt */
  own: boolean
}

/** Mitglied in der Teilnehmerliste eines Projekts */
export interface MemberSummary {
  id: string
  email: string
  role: Role
  status: UserStatus | ''
  firstName: string
  lastName: string
  birthDate: string
  guest: boolean
  joinedAt: string
  sanctions: Sanction[]
}

export interface ProjectSearchResult {
  slug: string
  name: string
  region: string
  description: string
  hasLogo: boolean
  member: boolean
}

/** Öffentliche Angaben eines Projekts (Login über den Projekt-Link) */
export interface PublicProject {
  slug: string
  name: string
  region: string
  hasLogo: boolean
}

/** Geteilte Verwarnungen und Sperren mit einem Partnerprojekt */
export interface ProjectShare {
  partnerSlug: string
  partnerName: string
  /** dieses Projekt hat angefragt, der Partner muss einwilligen */
  outgoing: boolean
  state: 'ausstehend' | 'eingewilligt' | 'abgelehnt'
  consentedAt: string | null
}

export interface AdminProject {
  slug: string
  name: string
  region: string
  description: string
  hasLogo: boolean
  pending: boolean
  memberCount: number
  admins: string[]
  shares: ProjectShare[]
  createdAt: string
}

export interface AdminUser {
  id: string
  email: string
  firstName: string
  lastName: string
  birthDate: string
  guest: boolean
  globalAdmin: boolean
  projects: string[]
}

export interface AuditRecord {
  id: number
  at: string
  actorId: string | null
  actorName: string
  projectName: string | null
  action: string
  targetId: string | null
  targetName: string
  details: Record<string, unknown>
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
  sanctions: Sanction[]
}

export interface CheckinResult {
  checkinId: number
  person: User
  status: UserStatus | ''
  registered: boolean
  alreadyCheckedIn: boolean
  checkedInAt: string | null
  /** geltende Verwarnungen und Sperren, als Meldung anzeigen */
  sanctions: Sanction[]
  /** gesperrt, nicht eingecheckt */
  blocked: boolean
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

/** Rollen und Funktionen, die Notfallkontakte einsehen dürfen (Administration des Projekts). Admins immer. */
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

/** API mit Pfad-Präfix, z.B. "/p/aadorf" für die Routen eines Projekts */
export function apiWithPrefix(prefix: string) {
  return {
    get: <T>(path: string) => request<T>('GET', prefix + path),
    post: <T>(path: string, body: unknown = {}) => request<T>('POST', prefix + path, body),
    put: <T>(path: string, body: unknown) => request<T>('PUT', prefix + path, body),
    del: <T = void>(path: string) => request<T>('DELETE', prefix + path),
  }
}

export type Api = ReturnType<typeof apiWithPrefix>

/** Routen ohne Projekt (Person, Projekte suchen, globale Administration) */
export const api = apiWithPrefix('')

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'Unbekannter Fehler'
}
