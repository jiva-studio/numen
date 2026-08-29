import { describe, expect, it } from 'vitest'

import { days, fits, names, ROWS, weighs } from './heatmap'

describe('how much of a year fits', () => {
  // A wide window shows more of the year rather than the same weeks drawn
  // larger, and a narrow one shows fewer rather than a grid marooned in the
  // middle of empty room.
  it('takes as many columns as the room holds', () => {
    const cell = 10
    const gap = 2
    expect(fits({ width: 120, cell, gap }).columns).toBe(10)
    expect(fits({ width: 60, cell, gap }).columns).toBe(5)
    expect(fits({ width: 600, cell, gap }).columns).toBe(50)
  })

  it('draws a cell no larger than it is asked to, whatever the room', () => {
    for (const width of [40, 200, 1000, 4000]) {
      expect(fits({ width, cell: 11, gap: 3 }).cell).toBe(11)
    }
  })

  it('spreads what is left over between the cells, so the grid meets both edges', () => {
    const room = fits({ width: 200, cell: 10, gap: 2 })
    const drawn = room.columns * 10 + (room.columns - 1) * room.gap
    expect(drawn).toBeCloseTo(200, 5)
    expect(room.gap).toBeGreaterThanOrEqual(2)
  })

  it('holds one column where there is room for none', () => {
    expect(fits({ width: 0, cell: 10, gap: 2 }).columns).toBe(1)
    expect(fits({ width: 4, cell: 10, gap: 2 }).columns).toBe(1)
  })
})

describe('the days a grid draws', () => {
  const did = new Map<string, number>()

  it('is a whole week to a column', () => {
    expect(days(8, new Date('2026-08-29T12:00:00'), did)).toHaveLength(8 * ROWS)
    expect(days(1, new Date('2026-08-29T12:00:00'), did)).toHaveLength(ROWS)
  })

  // Today stands in the last column, and the grid runs back from it, so a
  // person reads the year left to right and ends where they are.
  it('ends on the week today stands in', () => {
    // Saturday.
    const shown = days(4, new Date('2026-08-29T12:00:00'), did)
    const today = shown.filter((one) => one.today)

    expect(today).toHaveLength(1)
    expect(shown.indexOf(today[0]!)).toBeGreaterThan(shown.length - ROWS - 1)
    expect(shown[0]!.day < today[0]!.day).toBe(true)
  })

  it('reads what was done on each day it draws', () => {
    const on = new Date('2026-08-29T12:00:00')
    const counted = new Map([
      [names(on), 12],
      ['2026-08-28', 60],
    ])
    const shown = days(4, on, counted)

    expect(shown.find((one) => one.today)?.did).toBe(12)
    expect(shown.find((one) => one.day === '2026-08-28')?.weight).toBe(4)
    expect(shown.find((one) => one.day === '2026-08-27')?.did).toBe(0)
  })

  it('draws no days where there is room for no column', () => {
    expect(days(0, new Date('2026-08-29T12:00:00'), did)).toHaveLength(0)
  })
})

describe('how dark a day is drawn', () => {
  // A person who answered five cards did sit down, and the grid says so as
  // plainly as it says a day of fifty.
  it('is nothing for a day nobody answered on, and rises with what was done', () => {
    expect(weighs(0)).toBe(0)
    expect(weighs(1)).toBe(1)
    expect(weighs(4)).toBe(1)
    expect(weighs(5)).toBe(2)
    expect(weighs(19)).toBe(2)
    expect(weighs(20)).toBe(3)
    expect(weighs(49)).toBe(3)
    expect(weighs(50)).toBe(4)
    expect(weighs(5000)).toBe(4)
  })
})

describe('the name of a day', () => {
  it('is the year, the month and the day, each padded', () => {
    expect(names(new Date(2026, 0, 5))).toBe('2026-01-05')
    expect(names(new Date(2026, 11, 31))).toBe('2026-12-31')
  })
})
