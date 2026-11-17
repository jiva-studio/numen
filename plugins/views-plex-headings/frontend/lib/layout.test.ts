/** Where the list hangs, how many rows stand in it and how far a wheel scrolls it. */
import { describe, expect, it } from 'vitest'
import { getFurthestOffset, getListLayout, getListStart, scrollBy, type ListLayout } from './layout'
import { getMeasures } from './measures'

const MEASURES = { rowHeight: 20, indentStep: 10, margin: 20, pad: 5 }
const ROOM = { width: 1000, height: 600 }
const NODE = { x: 450, y: 280, width: 100, height: 40 }

const layoutOf = (overrides: Partial<Parameters<typeof getListLayout>[0]> = {}): ListLayout =>
  getListLayout({ count: 10, limit: 6, node: NODE, room: ROOM, measures: MEASURES, ...overrides })!

describe('the layout of the list', () => {
  it('is none for a note with no headings', () => {
    expect(getListLayout({ count: 0, limit: 6, node: NODE, room: ROOM, measures: MEASURES })).toBeNull()
  })

  it('holds the most rows the limit allows, where there is room below the node', () => {
    expect(layoutOf().visibleCount).toBe(6)
    expect(layoutOf({ limit: 3 }).visibleCount).toBe(3)
  })

  it('holds no more rows than there are headings', () => {
    expect(layoutOf({ count: 2 }).visibleCount).toBe(2)
  })

  it('is as tall as its rows and a pad above and under them', () => {
    expect(layoutOf().height).toBe(6 * 20 + 2 * 5)
  })

  it('holds only the rows the depth below the node holds, inside the margin of the window', () => {
    const lower = { ...NODE, y: 440 }
    const depth = ROOM.height - MEASURES.margin - (lower.y + lower.height)

    expect(layoutOf({ node: lower, limit: 12 }).visibleCount).toBe(
      Math.floor((depth - 2 * MEASURES.pad) / MEASURES.rowHeight),
    )
  })

  it('is none where not one row fits under the node', () => {
    expect(
      getListLayout({
        count: 3,
        limit: 6,
        node: { ...NODE, y: 550 },
        room: ROOM,
        measures: MEASURES,
      }),
    ).toBeNull()
  })

  it('takes its lengths from the type: a larger type, larger rows', () => {
    const designed = getMeasures(13)
    const larger = getMeasures(26)

    expect(larger.rowHeight).toBe(designed.rowHeight * 2)
    expect(larger.indentStep).toBe(designed.indentStep * 2)
    expect(larger.margin).toBe(designed.margin * 2)
  })
})

describe('the furthest the window on the rows scrolls', () => {
  it('is what is beyond the rows that stand', () => {
    expect(getFurthestOffset(10, layoutOf())).toBe(4)
  })

  it('is none where every row stands', () => {
    expect(getFurthestOffset(4, layoutOf({ count: 4 }))).toBe(0)
  })
})

describe('where the list begins', () => {
  it('centers it on the node', () => {
    expect(getListStart(layoutOf(), 200)).toBe(400)
  })

  it('moves it back inside the window where centered it would leave', () => {
    const near = layoutOf({ node: { ...NODE, x: 900 } })
    // A list of 200 keeps the margin of 20 from the right edge of the window, so it begins no further than 780.
    expect(getListStart(near, 200)).toBe(780)
  })

  it('moves it the other way at the other edge of the window', () => {
    const near = layoutOf({ node: { ...NODE, x: 0 } })
    expect(getListStart(near, 200)).toBe(MEASURES.margin)
  })
})

describe('what a wheel scrolls', () => {
  const layout = layoutOf()

  it('is a row for each row of pixels it moved', () => {
    expect(scrollBy(layout, { delta: 45, mode: 0 }, 0)).toStrictEqual({ by: 2, left: 5 })
    expect(scrollBy(layout, { delta: -45, mode: 0 }, 0)).toStrictEqual({ by: -2, left: -5 })
  })

  it('adds what was left over from the last wheel', () => {
    expect(scrollBy(layout, { delta: 15, mode: 0 }, 10)).toStrictEqual({ by: 1, left: 5 })
  })

  it('is a row for each line the wheel moved', () => {
    expect(scrollBy(layout, { delta: 3, mode: 1 }, 0)).toStrictEqual({ by: 3, left: 0 })
  })

  it('is the rows that stand for each page the wheel moved', () => {
    expect(scrollBy(layout, { delta: 1, mode: 2 }, 0)).toStrictEqual({ by: 6, left: 0 })
  })
})
