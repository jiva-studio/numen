/** The side a point asks for, and the slot in a strip of tabs. */
import { describe, expect, it } from 'vitest'
import { caretAt, edgeOf, overlayFor, sideAt, slotAt, DEFAULT_DROP } from './drop'

const BOX = { x: 0, y: 0, width: 800, height: 400 }

describe('sideAt', () => {
  it('lands in the middle away from every edge', () => {
    expect(sideAt({ x: 400, y: 200 }, BOX)).toBe('center')
  })

  it.each([
    ['left', { x: 20, y: 200 }],
    ['right', { x: 780, y: 200 }],
    ['top', { x: 400, y: 10 }],
    ['bottom', { x: 400, y: 390 }],
  ] as const)('reads %s', (side, point) => {
    expect(sideAt(point, BOX)).toBe(side)
  })

  it('gives a corner to the edge it is deeper inside', () => {
    // Twenty into a zone of eighty across, ten into a zone of eighty down.
    expect(sideAt({ x: 20, y: 10 }, BOX)).toBe('top')
    expect(sideAt({ x: 10, y: 20 }, BOX)).toBe('left')
  })

  it('measures the zone from the box, up to the limit', () => {
    const wide = { x: 0, y: 0, width: 4000, height: 400 }
    const share = wide.width * DEFAULT_DROP.share

    expect(share).toBeGreaterThan(DEFAULT_DROP.limit)
    expect(sideAt({ x: DEFAULT_DROP.limit + 1, y: 200 }, wide)).toBe('center')
    expect(sideAt({ x: DEFAULT_DROP.limit - 1, y: 200 }, wide)).toBe('left')
  })

  it('reads a box that is offset from the origin', () => {
    const offset = { x: 100, y: 50, width: 800, height: 400 }
    expect(sideAt({ x: 120, y: 250 }, offset)).toBe('left')
    expect(sideAt({ x: 500, y: 250 }, offset)).toBe('center')
  })

  it('takes a narrower zone when asked', () => {
    expect(sideAt({ x: 20, y: 200 }, BOX, { share: 0.01 })).toBe('center')
  })
})

describe('overlayFor', () => {
  it('shows the half a side would take', () => {
    expect(overlayFor('right', BOX)).toStrictEqual({ x: 400, y: 0, width: 400, height: 400 })
    expect(overlayFor('bottom', BOX)).toStrictEqual({ x: 0, y: 200, width: 800, height: 200 })
  })

  it('shows the whole pane for the middle', () => {
    expect(overlayFor('center', BOX)).toStrictEqual(BOX)
  })
})

describe('slotAt', () => {
  const tabs = [
    { x: 0, y: 0, width: 100, height: 30 },
    { x: 100, y: 0, width: 100, height: 30 },
    { x: 200, y: 0, width: 100, height: 30 },
  ]

  it('counts the gaps between tabs', () => {
    expect(slotAt(10, tabs)).toBe(0)
    expect(slotAt(60, tabs)).toBe(1)
    expect(slotAt(160, tabs)).toBe(2)
    expect(slotAt(400, tabs)).toBe(3)
  })

  it('passes a tab at its middle', () => {
    expect(slotAt(49, tabs)).toBe(0)
    expect(slotAt(51, tabs)).toBe(1)
  })

  it('has one place in an empty strip', () => {
    expect(slotAt(0, [])).toBe(0)
  })
})

describe('caretAt', () => {
  const strip = { x: 0, y: 0, width: 300, height: 30 }
  const tabs = [
    { x: 0, y: 0, width: 100, height: 30 },
    { x: 100, y: 0, width: 100, height: 30 },
    { x: 200, y: 0, width: 100, height: 30 },
  ]

  it('stands at the leading edge of the tab it comes before', () => {
    expect(caretAt(1, tabs, strip)).toStrictEqual({ x: 100, y: 0, width: 0, height: 30 })
  })

  it('stands past the last tab for the place after it', () => {
    expect(caretAt(3, tabs, strip)).toStrictEqual({ x: 300, y: 0, width: 0, height: 30 })
  })

  it('takes the whole strip where there is no tab to stand beside', () => {
    expect(caretAt(0, [], strip)).toStrictEqual(strip)
  })
})

describe('edgeOf', () => {
  it.each([
    ['left', { x: 4, y: 200 }],
    ['right', { x: 796, y: 200 }],
    ['top', { x: 400, y: 4 }],
    ['bottom', { x: 400, y: 396 }],
  ])('reaches the %s edge', (side, point) => {
    expect(edgeOf(point, BOX, 22)).toBe(side)
  })

  it('reaches no edge from the middle', () => {
    expect(edgeOf({ x: 400, y: 200 }, BOX, 22)).toBeNull()
  })

  it('gives a corner to the edge it stands nearer', () => {
    expect(edgeOf({ x: 3, y: 10 }, BOX, 22)).toBe('left')
    expect(edgeOf({ x: 10, y: 3 }, BOX, 22)).toBe('top')
  })

  it('reads a box that does not start at the origin', () => {
    const box = { x: 100, y: 50, width: 200, height: 100 }
    expect(edgeOf({ x: 104, y: 100 }, box, 22)).toBe('left')
    expect(edgeOf({ x: 200, y: 100 }, box, 22)).toBeNull()
  })

  it('reaches nothing at no reach at all', () => {
    expect(edgeOf({ x: 0, y: 200 }, BOX, 0)).toBeNull()
  })
})
