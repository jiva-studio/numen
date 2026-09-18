import { describe, expect, it } from 'vitest'

import { getCells, getDays, getGridHeight, getWeight, measureGrid, NOTHING, ROWS } from './heatmap'
import { getDayName } from '@/shared/lib/day'
import type { Tally } from './heatmap'

describe('how much of a year fits', () => {
  // A wide window shows more of the year rather than the same weeks drawn
  // larger, and a narrow one shows fewer rather than a grid marooned in the
  // middle of empty room.
  it('takes as many columns as the room holds', () => {
    const cell = 10
    const gap = 2
    expect(measureGrid({ width: 120, cell, gap }).columns).toBe(10)
    expect(measureGrid({ width: 60, cell, gap }).columns).toBe(5)
    expect(measureGrid({ width: 600, cell, gap }).columns).toBe(50)
  })

  it('draws a cell no larger than it is asked to, whatever the room', () => {
    for (const width of [40, 200, 1000, 4000]) {
      expect(measureGrid({ width, cell: 11, gap: 3 }).cell).toBe(11)
    }
  })

  it('countSpreads what is left over between the cells, so the grid meets both edges', () => {
    const room = measureGrid({ width: 200, cell: 10, gap: 2 })
    const drawn = room.columns * 10 + (room.columns - 1) * room.gap
    expect(drawn).toBeCloseTo(200, 5)
    expect(room.gap).toBeGreaterThanOrEqual(2)
  })

  it('holds one column where there is room for none', () => {
    expect(measureGrid({ width: 0, cell: 10, gap: 2 }).columns).toBe(1)
    expect(measureGrid({ width: 4, cell: 10, gap: 2 }).columns).toBe(1)
  })
})

/** The days a person answered on, as many cards as each says. */
const createTallies = (days: [string, number][]) =>
  new Map(
    days.map(([day, answered]) => [
      day,
      { ...NOTHING, answered, good: answered, asked: answered, recalled: answered },
    ]),
  )

describe('the days a grid draws', () => {
  const did = new Map<string, Tally>()

  it('is a whole week to a column', () => {
    expect(getDays(8, new Date('2026-08-29T12:00:00'), did)).toHaveLength(8 * ROWS)
    expect(getDays(1, new Date('2026-08-29T12:00:00'), did)).toHaveLength(ROWS)
  })

  // The weeks behind run up to the one a person is in, and a few weeks of what
  // is coming stand after it, so the grid says what is ahead as well.
  it('keeps room after today for what is still to come', () => {
    // Saturday.
    const now = new Date('2026-08-29T12:00:00')
    const shown = getDays(20, now, did)
    const today = shown.filter((one) => one.isToday)

    expect(today).toHaveLength(1)
    expect(shown[0]!.day < today[0]!.day).toBe(true)

    const after = shown.slice(shown.indexOf(today[0]!) + 1)
    expect(after.length).toBeGreaterThan(ROWS * 3)
    for (const one of after) {
      expect(one.isAhead).toBe(true)
    }
    for (const one of shown.slice(0, shown.indexOf(today[0]!) + 1)) {
      expect(one.isAhead).toBe(false)
    }
  })

  // The grid is as wide as the room it was given: a week to every column.
  it('draws a week for every column it was given', () => {
    const now = new Date('2026-08-29T12:00:00')
    const shown = getDays(30, now, createTallies([['2026-08-28', 3]]))

    expect(shown).toHaveLength(30 * ROWS)
    expect(shown.some((one) => one.isToday)).toBe(true)
  })

  // Nobody wants years of empty squares from before they ever sat down. The
  // grid opens on the week they began in, and the room past what they have yet
  // done stretches out to the right.
  it('opens on the week a person began in', () => {
    const now = new Date('2026-08-29T12:00:00')
    const shown = getDays(30, now, createTallies([['2026-08-28', 3]]))

    // The Monday of that week.
    expect(shown[0]!.day).toBe('2026-08-24')
    expect(shown[shown.length - 1]!.isAhead).toBe(true)
  })

  // Once they have been here longer than the width holds, the oldest weeks
  // fall off the left and the grid ends on what is still to come.
  it('lets the oldest weeks go once there are more than it holds', () => {
    const now = new Date('2026-08-29T12:00:00')
    const long = createTallies([
      ['2024-01-01', 5],
      ['2026-08-28', 3],
    ])
    const shown = getDays(12, now, long)

    expect(shown[0]!.day > '2024-01-01').toBe(true)
    expect(shown.some((one) => one.isToday)).toBe(true)
    expect(shown[shown.length - 1]!.isAhead).toBe(true)
  })

  // A vault whose cards are all still ahead has a beginning too.
  it('opens on this week for a vault with nothing behind it', () => {
    const now = new Date('2026-08-29T12:00:00')
    const shown = getDays(30, now, new Map(), new Map([['2026-09-03', 8]]))

    expect(shown[0]!.day).toBe('2026-08-24')
  })

  it('gives the room to what is behind where there is little of it', () => {
    const shown = getDays(1, new Date('2026-08-29T12:00:00'), did)
    expect(shown).toHaveLength(ROWS)
    expect(shown.some((one) => one.isToday)).toBe(true)
  })

  // A day still to come holds what falls on it, and a day behind holds what was
  // answered on it. Today holds what was answered: that is the number a person
  // is adding to.
  it('reads what is coming for the days still to come', () => {
    const now = new Date('2026-08-29T12:00:00')
    const done = createTallies([
      ['2026-08-29', 4],
      ['2026-09-02', 99],
    ])
    const coming = new Map([
      ['2026-08-29', 99],
      ['2026-09-02', 7],
    ])
    const shown = getDays(12, now, done, coming)

    expect(shown.find((one) => one.isToday)?.did).toBe(4)
    const later = shown.find((one) => one.day === '2026-09-02')
    expect(later?.did).toBe(7)
    expect(later?.isAhead).toBe(true)
  })

  it('reads what was done on each day it draws', () => {
    const on = new Date('2026-08-29T12:00:00')
    const counted = createTallies([
      [getDayName(on), 12],
      ['2026-08-28', 60],
    ])
    const shown = getDays(4, on, counted)

    expect(shown.find((one) => one.isToday)?.did).toBe(12)
    expect(shown.find((one) => one.day === '2026-08-28')?.weight).toBe(4)
    expect(shown.find((one) => one.day === '2026-08-27')?.did).toBe(0)
  })

  it('draws no days where there is room for no column', () => {
    expect(getDays(0, new Date('2026-08-29T12:00:00'), did)).toHaveLength(0)
  })
})

describe('how dark a day is drawn', () => {
  // A person who answered five cards did sit down, and the grid says so as
  // plainly as it says a day of fifty.
  it('is nothing for a day nobody answered on, and rises with what was done', () => {
    expect(getWeight(0)).toBe(0)
    expect(getWeight(1)).toBe(1)
    expect(getWeight(4)).toBe(1)
    expect(getWeight(5)).toBe(2)
    expect(getWeight(19)).toBe(2)
    expect(getWeight(20)).toBe(3)
    expect(getWeight(49)).toBe(3)
    expect(getWeight(50)).toBe(4)
    expect(getWeight(5000)).toBe(4)
  })
})

describe('where the days are drawn', () => {
  const grid = { cell: 10, gap: 2 }

  it('stands seven rows tall, with no gap past the last', () => {
    expect(getGridHeight(grid)).toBe(7 * 12 - 2)
  })

  it('fills a column before it begins the next', () => {
    const days = getDays(2, new Date('2026-01-31T12:00:00Z'), new Map(), new Map())
    const cells = getCells(grid, days)

    expect(cells.slice(0, 7).map((one) => one.x)).toStrictEqual([0, 0, 0, 0, 0, 0, 0])
    expect(cells.slice(0, 7).map((one) => one.y)).toStrictEqual([0, 12, 24, 36, 48, 60, 72])
    expect(cells[7]?.x).toBe(12)
    expect(cells[7]?.y).toBe(0)
  })

  it('draws every cell the size the grid measured', () => {
    const days = getDays(1, new Date('2026-01-31T12:00:00Z'), new Map(), new Map())

    for (const cell of getCells(grid, days)) expect(cell.size).toBe(10)
  })
})
