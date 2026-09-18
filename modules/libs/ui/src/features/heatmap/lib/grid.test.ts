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
      for (const one of getDays(columns, now, new Map(), new Map())) {
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

  // A grid drawn wider than the room the stretch was asked for draws days
  // nobody asked about, and they come back empty. The caller passes the room
  // the page has, which is what bounds the grid.
  it('leaves days out where the grid is drawn wider than it was asked for', () => {
    const asked = getStretch({ width: 640, cell: 11, gap: 3 }, now)
    const { columns } = measureGrid({ width: 1600, cell: 11, gap: 3 })
    const outside = getDays(columns, now, new Map(), new Map()).filter(
      (one) => one.day < asked.from || one.day > asked.to,
    )
    expect(outside.length).toBeGreaterThan(0)
  })

  // A wider grid asks about more time, and never about less.
  it('grows with the room there is', () => {
    const narrow = getStretch({ width: 200, cell: 11, gap: 3 }, now)
    const wide = getStretch({ width: 1600, cell: 11, gap: 3 }, now)
    expect(wide.from < narrow.from).toBe(true)
    expect(wide.to >= narrow.to).toBe(true)
  })
})
