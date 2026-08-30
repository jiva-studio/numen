import { describe, expect, it } from 'vitest'

import { days, fits, marks, names, NOTHING, ROWS, weighs } from './heatmap'
import type { Tally } from './heatmap'

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

/** answeredOn is days a person answered on, as many cards as each says. */
const answeredOn = (days: [string, number][]) =>
  new Map(
    days.map(([day, answered]) => [
      day,
      { ...NOTHING, answered, good: answered, asked: answered, recalled: answered },
    ]),
  )

describe('the days a grid draws', () => {
  const did = new Map<string, Tally>()

  it('is a whole week to a column', () => {
    expect(days(8, new Date('2026-08-29T12:00:00'), did)).toHaveLength(8 * ROWS)
    expect(days(1, new Date('2026-08-29T12:00:00'), did)).toHaveLength(ROWS)
  })

  // The weeks behind run up to the one a person is in, and a few weeks of what
  // is coming stand after it, so the grid says what is ahead as well.
  it('keeps room after today for what is still to come', () => {
    // Saturday.
    const now = new Date('2026-08-29T12:00:00')
    const shown = days(20, now, did)
    const today = shown.filter((one) => one.today)

    expect(today).toHaveLength(1)
    expect(shown[0]!.day < today[0]!.day).toBe(true)

    const after = shown.slice(shown.indexOf(today[0]!) + 1)
    expect(after.length).toBeGreaterThan(ROWS * 3)
    for (const one of after) {
      expect(one.ahead).toBe(true)
    }
    for (const one of shown.slice(0, shown.indexOf(today[0]!) + 1)) {
      expect(one.ahead).toBe(false)
    }
  })

  // The grid is as wide as the room it was given, so it runs back from the
  // last week it draws for as many weeks as it has columns.
  it('draws a week for every column it was given', () => {
    const now = new Date('2026-08-29T12:00:00')
    const shown = days(30, now, answeredOn([['2026-08-28', 3]]))

    expect(shown).toHaveLength(30 * ROWS)
    expect(shown.some((one) => one.today)).toBe(true)
    expect(shown[shown.length - 1]!.ahead).toBe(true)
  })

  it('gives the room to what is behind where there is little of it', () => {
    const shown = days(1, new Date('2026-08-29T12:00:00'), did)
    expect(shown).toHaveLength(ROWS)
    expect(shown.some((one) => one.today)).toBe(true)
  })

  // A day still to come holds what falls on it, and a day behind holds what was
  // answered on it. Today holds what was answered: that is the number a person
  // is adding to.
  it('reads what is coming for the days still to come', () => {
    const now = new Date('2026-08-29T12:00:00')
    const done = answeredOn([
      ['2026-08-29', 4],
      ['2026-09-02', 99],
    ])
    const coming = new Map([
      ['2026-08-29', 99],
      ['2026-09-02', 7],
    ])
    const shown = days(12, now, done, coming)

    expect(shown.find((one) => one.today)?.did).toBe(4)
    const later = shown.find((one) => one.day === '2026-09-02')
    expect(later?.did).toBe(7)
    expect(later?.ahead).toBe(true)
  })

  it('reads what was done on each day it draws', () => {
    const on = new Date('2026-08-29T12:00:00')
    const counted = answeredOn([
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

describe('the months over the grid', () => {
  // Two labels over neighbouring columns run into one another and read as one
  // word, and the one carrying a year is the wider of the two.
  it('leaves room between them, and more after one carrying a year', () => {
    const shown = days(80, new Date('2026-08-29T12:00:00'), new Map())
    const said = marks(shown, 3, 6)

    for (let at = 1; at < said.length; at += 1) {
      const before = said[at - 1]!
      const room = before.year ? 6 : 3
      expect(said[at]!.column - before.column).toBeGreaterThanOrEqual(room)
    }
  })

  it('says the year where the year turns', () => {
    const shown = days(80, new Date('2026-08-29T12:00:00'), new Map())
    const said = marks(shown)

    expect(said.filter((one) => one.year).length).toBeGreaterThan(0)
    for (const one of said) {
      expect(one.day.endsWith('-01') || one.day > '2020-01-01').toBe(true)
    }
  })

  it('says nothing over a grid of nothing', () => {
    expect(marks([])).toHaveLength(0)
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
