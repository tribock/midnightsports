import { describe, expect, it } from 'vitest'
import { age, formatShortDate, todayISO } from './format'

describe('age', () => {
  it('zählt erst ab dem Geburtstag', () => {
    expect(age('2010-11-07', new Date(2026, 10, 6))).toBe(15)
    expect(age('2010-11-07', new Date(2026, 10, 7))).toBe(16)
  })

  it('ohne Geburtsdatum', () => {
    expect(age('')).toBeNull()
  })
})

describe('formatShortDate', () => {
  it('formatiert im Schweizer Format', () => {
    expect(formatShortDate('2026-11-07')).toBe('07.11.2026')
  })
})

describe('todayISO', () => {
  it('verwendet die lokale Zeit', () => {
    expect(todayISO(new Date(2026, 0, 5, 23, 30))).toBe('2026-01-05')
  })
})
