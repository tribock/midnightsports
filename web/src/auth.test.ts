import { describe, expect, it } from 'vitest'
import type { Role } from './api'
import { isAdmin, isLeader, isProjectLead } from './auth'

describe('Rollen', () => {
  const as = (role: Role) => ({ role })

  it('Admins dürfen alles', () => {
    expect(isAdmin(as('admin'))).toBe(true)
    expect(isProjectLead(as('admin'))).toBe(true)
    expect(isLeader(as('admin'))).toBe(true)
  })

  it('Projektleiter: Events, Berichte und Löhne, keine Administration', () => {
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
