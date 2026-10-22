/** The side a point asks for, and the slot in a strip of tabs. */
import { describe, expect, it } from 'vitest'
import {
  getCaretRect,
  findEdgeSide,
  getOverlay,
  getRect,
  getSide,
  getSlot,
  DEFAULT_DROP,
} from './drop'

const BOX = { x: 0, y: 0, width: 800, height: 400 }

describe('getSide', () => {
  it('lands in the middle away from every edge', () => {
    expect(getSide({ x: 400, y: 200 }, BOX)).toBe('center')
  })

  it.each([
    ['left', { x: 20, y: 200 }],
    ['right', { x: 780, y: 200 }],
    ['top', { x: 400, y: 10 }],
    ['bottom', { x: 400, y: 390 }],
  ] as const)('reads %s', (side, point) => {
    expect(getSide(point, BOX)).toBe(side)
  })

  it('gives a corner to the edge it is deeper inside', () => {
    // Twenty into a zone of eighty across, ten into a zone of eighty down.
    expect(getSide({ x: 20, y: 10 }, BOX)).toBe('top')
    expect(getSide({ x: 10, y: 20 }, BOX)).toBe('left')
  })

  it('measures the zone from the box, up to the limit', () => {
    const wideBox = { x: 0, y: 0, width: 4000, height: 400 }
    const share = wideBox.width * DEFAULT_DROP.share

    expect(share).toBeGreaterThan(DEFAULT_DROP.limit)
    expect(getSide({ x: DEFAULT_DROP.limit + 1, y: 200 }, wideBox)).toBe('center')
    expect(getSide({ x: DEFAULT_DROP.limit - 1, y: 200 }, wideBox)).toBe('left')
  })

  it('reads a box that is offset from the origin', () => {
    const offset = { x: 100, y: 50, width: 800, height: 400 }
    expect(getSide({ x: 120, y: 250 }, offset)).toBe('left')
    expect(getSide({ x: 500, y: 250 }, offset)).toBe('center')
  })

  it('measures the zone down from the box, up to the limit', () => {
    const tall = { x: 0, y: 0, width: 400, height: 4000 }

    expect(getSide({ x: 200, y: DEFAULT_DROP.limit + 1 }, tall)).toBe('center')
    expect(getSide({ x: 200, y: DEFAULT_DROP.limit - 1 }, tall)).toBe('top')
  })

  it('takes a fifth of the height for the zone down', () => {
    expect(getSide({ x: 400, y: 85 }, BOX)).toBe('center')
    expect(getSide({ x: 400, y: 75 }, BOX)).toBe('top')
  })

  it('takes a fifth of a small box for the zone', () => {
    const small = { x: 0, y: 0, width: 40, height: 40 }

    expect(getSide({ x: 20, y: 20 }, small)).toBe('center')
    expect(getSide({ x: 20, y: 7 }, small)).toBe('top')
  })

  it('catches nothing in a zone of no size', () => {
    const none = { share: 0 }

    expect(getSide({ x: -5, y: 200 }, BOX, none)).toBe('center')
    expect(getSide({ x: 805, y: 200 }, BOX, none)).toBe('center')
    expect(getSide({ x: 400, y: -5 }, BOX, none)).toBe('center')
    expect(getSide({ x: 400, y: 405 }, BOX, none)).toBe('center')
  })

  it('leaves the edge of a zone to the middle', () => {
    expect(getSide({ x: 96, y: 200 }, BOX)).toBe('center')
    expect(getSide({ x: 704, y: 200 }, BOX)).toBe('center')
    expect(getSide({ x: 400, y: 80 }, BOX)).toBe('center')
    expect(getSide({ x: 400, y: 320 }, BOX)).toBe('center')
  })

  it('gives a corner equally deep to the first edge', () => {
    expect(getSide({ x: 12, y: 10 }, BOX)).toBe('left')
  })

  it('takes a narrower zone when requested', () => {
    expect(getSide({ x: 20, y: 200 }, BOX, { share: 0.01 })).toBe('center')
  })
})

describe('getOverlay', () => {
  it('shows the half a side would take', () => {
    expect(getOverlay('right', BOX)).toStrictEqual({ x: 400, y: 0, width: 400, height: 400 })
    expect(getOverlay('bottom', BOX)).toStrictEqual({ x: 0, y: 200, width: 800, height: 200 })
  })

  it('shows the half a side on the leading edge would take', () => {
    expect(getOverlay('left', BOX)).toStrictEqual({ x: 0, y: 0, width: 400, height: 400 })
    expect(getOverlay('top', BOX)).toStrictEqual({ x: 0, y: 0, width: 800, height: 200 })
  })

  it('starts from the box it is given', () => {
    const offset = { x: 100, y: 50, width: 800, height: 400 }

    expect(getOverlay('left', offset)).toStrictEqual({ x: 100, y: 50, width: 400, height: 400 })
    expect(getOverlay('top', offset)).toStrictEqual({ x: 100, y: 50, width: 800, height: 200 })
  })

  it('shows the whole pane for the middle', () => {
    expect(getOverlay('center', BOX)).toStrictEqual(BOX)
  })
})

describe('getSlot', () => {
  const tabs = [
    { x: 0, y: 0, width: 100, height: 30 },
    { x: 100, y: 0, width: 100, height: 30 },
    { x: 200, y: 0, width: 100, height: 30 },
  ]

  it('counts the gaps between tabs', () => {
    expect(getSlot(10, tabs)).toBe(0)
    expect(getSlot(60, tabs)).toBe(1)
    expect(getSlot(160, tabs)).toBe(2)
    expect(getSlot(400, tabs)).toBe(3)
  })

  it('passes a tab at its middle', () => {
    expect(getSlot(49, tabs)).toBe(0)
    expect(getSlot(51, tabs)).toBe(1)
  })

  it('passes a tab once the point reaches its middle', () => {
    expect(getSlot(50, tabs)).toBe(1)
    expect(getSlot(150, tabs)).toBe(2)
  })

  it('has one place in an empty strip', () => {
    expect(getSlot(0, [])).toBe(0)
  })
})

describe('getCaretRect', () => {
  const strip = { x: 0, y: 0, width: 300, height: 30 }
  const tabs = [
    { x: 0, y: 0, width: 100, height: 30 },
    { x: 100, y: 0, width: 100, height: 30 },
    { x: 200, y: 0, width: 100, height: 30 },
  ]

  it('remains at the leading edge of the tab it arrives before', () => {
    expect(getCaretRect(1, tabs, strip)).toStrictEqual({ x: 100, y: 0, width: 0, height: 30 })
  })

  it('remains past the last tab for the place after it', () => {
    expect(getCaretRect(3, tabs, strip)).toStrictEqual({ x: 300, y: 0, width: 0, height: 30 })
  })

  it('takes the whole strip where there is no tab to stand alongside', () => {
    expect(getCaretRect(0, [], strip)).toStrictEqual(strip)
  })
})

describe('findEdgeSide', () => {
  it.each([
    ['left', { x: 4, y: 200 }],
    ['right', { x: 796, y: 200 }],
    ['top', { x: 400, y: 4 }],
    ['bottom', { x: 400, y: 396 }],
  ])('reaches the %s edge', (side, point) => {
    expect(findEdgeSide(point, BOX, 22)).toBe(side)
  })

  it('measures the top edge from the box', () => {
    const box = { x: 100, y: 50, width: 200, height: 100 }
    expect(findEdgeSide({ x: 200, y: 54 }, box, 22)).toBe('top')
  })

  it('reaches no edge from the middle', () => {
    expect(findEdgeSide({ x: 400, y: 200 }, BOX, 22)).toBeNull()
  })

  it('gives a corner to the edge it remains nearer', () => {
    expect(findEdgeSide({ x: 3, y: 10 }, BOX, 22)).toBe('left')
    expect(findEdgeSide({ x: 10, y: 3 }, BOX, 22)).toBe('top')
  })

  it('reads a box that does not start at the origin', () => {
    const box = { x: 100, y: 50, width: 200, height: 100 }
    expect(findEdgeSide({ x: 104, y: 100 }, box, 22)).toBe('left')
    expect(findEdgeSide({ x: 200, y: 100 }, box, 22)).toBeNull()
  })

  it('reaches nothing at no reach at all', () => {
    expect(findEdgeSide({ x: 0, y: 200 }, BOX, 0)).toBeNull()
  })
})

describe('getRect', () => {
  it('reads what an element takes up on screen', () => {
    const element = {
      getBoundingClientRect: () => ({ left: 10, top: 20, width: 30, height: 40 }),
    } as unknown as Element

    expect(getRect(element)).toStrictEqual({ x: 10, y: 20, width: 30, height: 40 })
  })
})
