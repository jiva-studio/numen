/** The side a point asks for, and the slot in a strip of tabs. */
import { describe, expect, it } from 'vitest'
import { DEFAULT_DROP, overlayFor, sideAt, slotAt } from './drop'

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
