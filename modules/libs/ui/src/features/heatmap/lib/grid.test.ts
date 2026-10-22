/**
 * The room a grid is laid out in, and the stretch of time a grid that wide
 * draws.
 */
import { describe, expect, it } from 'vitest'

import { getStretch, measureGrid } from './grid'
import { getDays } from './heatmap'

describe('the stretch a grid asks about', () => {
  const now = new Date('2026-09-18T12:00:00Z')

  // Whatever room a grid ends up with, the days it draws stand inside what was
  // asked about for the widest room it could have had.
  it('holds every day a narrower grid would draw', () => {
    const widest = { width: 1600, cell: 11, gap: 3 }
    const stretch = getStretch(widest, now)

    for (const width of [120, 320, 640, 900, 1600]) {
      const { columns } = measureGrid({ ...widest, width })
      for (const one of getDays(columns, now, new Map())) {
        expect(one.day >= stretch.from && one.day <= stretch.to).toBe(true)
      }
    }
  })

  // It opens on a Monday and ends on a Sunday, because a column is a whole week.
  it('runs from a Monday to a Sunday', () => {
    const { from, to } = getStretch({ width: 640, cell: 11, gap: 3 }, now)
    expect(new Date(`${from}T12:00:00Z`).getUTCDay()).toBe(1)
    expect(new Date(`${to}T12:00:00Z`).getUTCDay()).toBe(0)
  })

  // nobody asked about, and they come back empty. The caller passes the room
  // the page has, which is what bounds the grid.
  it('leaves days out where the grid is drawn wider than it was requested for', () => {
    const request = getStretch({ width: 640, cell: 11, gap: 3 }, now)
    const { columns } = measureGrid({ width: 1600, cell: 11, gap: 3 })
    const outside = getDays(columns, now, new Map()).filter(
      (one) => one.day < request.from || one.day > request.to,
    )
    expect(outside.length).toBeGreaterThan(0)
  })

  // A wider grid asks about more time, and never about less.
  it('grows with the room there is', () => {
    const narrowStretch = getStretch({ width: 200, cell: 11, gap: 3 }, now)
    const wideStretch = getStretch({ width: 1600, cell: 11, gap: 3 }, now)
    expect(wideStretch.from < narrowStretch.from).toBe(true)
    expect(wideStretch.to >= narrowStretch.to).toBe(true)
  })
})

describe('the columns that fit a room', () => {
  const cell = 10
  const gap = 2

  it('keeps the gap it was given where one column fits', () => {
    expect(measureGrid({ width: 15, cell, gap })).toEqual({ columns: 1, cell, gap })
  })

  it('spreads what two columns leave over between them', () => {
    expect(measureGrid({ width: 30, cell, gap })).toEqual({ columns: 2, cell, gap: 10 })
  })
})

describe('the exact stretch of a grid', () => {
  const cell = 10
  const gap = 2
  // A Friday and a Sunday.
  const friday = new Date('2026-09-18T12:00:00')
  const sunday = new Date('2026-09-20T12:00:00')

  it('is the week of now for one column', () => {
    expect(getStretch({ width: 15, cell, gap }, friday)).toEqual({
      from: '2026-09-14',
      to: '2026-09-20',
    })
  })

  it('ends on the last week of the columns for two', () => {
    expect(getStretch({ width: 30, cell, gap }, friday)).toEqual({
      from: '2026-09-14',
      to: '2026-09-27',
    })
  })

  it('reaches back from four weeks ahead once there are more columns than weeks to come', () => {
    expect(getStretch({ width: 70, cell, gap }, friday)).toEqual({
      from: '2026-09-07',
      to: '2026-10-25',
    })
  })

  it('counts a Sunday as the last day of its own week', () => {
    expect(getStretch({ width: 15, cell, gap }, sunday)).toEqual({
      from: '2026-09-14',
      to: '2026-09-20',
    })
  })
})
