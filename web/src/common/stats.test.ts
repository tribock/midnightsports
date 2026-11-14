import { describe, expect, it } from 'vitest'
import { attendanceLabel } from './attendance'
import { lastMonths, perEvent, sum } from './stats'

describe('sum', () => {
  it('summiert einen Wert der Liste', () => {
    expect(sum([{ h: 1.5 }, { h: 2 }], (x) => x.h)).toBe(3.5)
    expect(sum([], () => 1)).toBe(0)
  })
})

describe('perEvent', () => {
  it('rundet auf eine Stelle', () => {
    expect(perEvent(257, 6)).toBe(42.8)
  })

  it('ohne Events 0', () => {
    expect(perEvent(10, 0)).toBe(0)
  })
})

describe('lastMonths', () => {
  it('8 Monate zurück bis heute', () => {
    expect(lastMonths(8, new Date(2026, 10, 14))).toEqual({ from: '2026-03-14', to: '2026-11-14' })
  })

  it('über den Jahreswechsel', () => {
    expect(lastMonths(8, new Date(2026, 1, 10))).toEqual({ from: '2025-06-10', to: '2026-02-10' })
  })
})

describe('attendanceLabel', () => {
  it('ohne Angabe nur, wenn vorhanden', () => {
    expect(attendanceLabel({ w: 3, m: 2, d: 0, other: 0 })).toBe('W 3 · M 2 · D 0')
    expect(attendanceLabel({ w: 3, m: 2, d: 1, other: 1 })).toBe('W 3 · M 2 · D 1 · ohne Angabe 1')
  })
})
