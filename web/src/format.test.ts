import { describe, expect, it } from 'vitest'
import { age, formatAddress, formatCHF, formatShortDate, parseDecimal, todayISO } from './format'

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

describe('formatCHF', () => {
  it('zeigt Franken mit zwei Nachkommastellen', () => {
    expect(formatCHF(40)).toMatch(/^CHF\s40\.00$/)
    expect(formatCHF(12.5)).toMatch(/^CHF\s12\.50$/)
  })
})

describe('parseDecimal', () => {
  it('akzeptiert Komma und Punkt', () => {
    expect(parseDecimal('3,5')).toBe(3.5)
    expect(parseDecimal(' 40 ')).toBe(40)
  })

  it('leere Eingabe ergibt null', () => {
    expect(parseDecimal('  ')).toBeNull()
  })
})

describe('formatAddress', () => {
  it('lässt fehlende Teile weg', () => {
    expect(formatAddress({ street: 'Hauptstrasse 1', zip: '8355', city: 'Aadorf' })).toBe('Hauptstrasse 1, 8355 Aadorf')
    expect(formatAddress({ street: '', zip: '8355', city: 'Aadorf' })).toBe('8355 Aadorf')
    expect(formatAddress({ street: '', zip: '', city: '' })).toBe('')
  })
})
