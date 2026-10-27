import { beforeAll, describe, expect, it, vi } from 'vitest'
import type { Role } from './api'
import { afterLogin, canViewMembers, isAdmin, isLeader, isProjectLead, rememberAfterLogin } from './auth'

describe('Rollen', () => {
  const as = (role: Role) => ({ role })

  it('Admins dürfen alles', () => {
    expect(isAdmin(as('admin'))).toBe(true)
    expect(isProjectLead(as('admin'))).toBe(true)
    expect(isLeader(as('admin'))).toBe(true)
  })

  it('Projektleiter: Events, Berichte, Löhne und Verwarnungen, keine Administration', () => {
    expect(isAdmin(as('projektleiter'))).toBe(false)
    expect(isProjectLead(as('projektleiter'))).toBe(true)
    expect(isLeader(as('projektleiter'))).toBe(true)
  })

  it('Mitarbeiter: nur Events', () => {
    expect(isProjectLead(as('mitarbeiter'))).toBe(false)
    expect(isLeader(as('mitarbeiter'))).toBe(true)
  })

  it('Teilnehmer und nicht angemeldet: nichts', () => {
    for (const user of [as('teilnehmer'), null]) {
      expect(isLeader(user)).toBe(false)
      expect(isProjectLead(user)).toBe(false)
      expect(isAdmin(user)).toBe(false)
    }
  })
})

describe('Teilnehmerliste', () => {
  it('Projektleiter, Admins und Personen mit Zugriff auf Notfallkontakte', () => {
    expect(canViewMembers({ role: 'projektleiter' })).toBe(true)
    expect(canViewMembers({ role: 'mitarbeiter', emergencyAccess: true })).toBe(true)
    expect(canViewMembers({ role: 'mitarbeiter', emergencyAccess: false })).toBe(false)
    expect(canViewMembers({ role: 'teilnehmer' })).toBe(false)
  })
})

describe('Ziel nach dem Login', () => {
  // Die Tests laufen ohne Browser: einfacher Speicher statt sessionStorage
  beforeAll(() => {
    const store = new Map<string, string>()
    vi.stubGlobal('sessionStorage', {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => store.set(k, v),
      removeItem: (k: string) => store.delete(k),
    })
  })

  it('merkt sich Pfade der App und entfernt sie beim Lesen', () => {
    rememberAfterLogin('/projekte/aadorf')
    expect(afterLogin()).toBe('/projekte/aadorf')
    expect(afterLogin(true)).toBe('/projekte/aadorf')
    expect(afterLogin()).toBeNull()
  })

  it('ignoriert fremde Adressen', () => {
    for (const path of ['https://evil.example', '//evil.example', 'javascript:alert(1)']) {
      rememberAfterLogin(path)
      expect(afterLogin(true)).toBeNull()
    }
  })
})
