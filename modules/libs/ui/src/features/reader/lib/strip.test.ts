import { describe, expect, it } from 'vitest'
import {
  CLOSEST,
  FURTHEST,
  GAP,
  NEARER,
  clampZoom,
  getPagesWithin,
  inFront,
  row,
  findPageScroll,
  getScrollTarget,
  getTurnTarget,
  type Page,
} from './strip'
import type { Size } from '@/shared/lib/geometry'

const VIEWPORT: Size = { width: 900, height: 800 }

/** Every page the same shape, the way a book is. */
const book = (pages: number): Page[] =>
  Array.from({ length: pages }, () => ({ width: 612, height: 792 }))

describe('the row a document makes', () => {
  it('draws a whole page in the viewport', () => {
    const stripLayout = row(book(4), VIEWPORT, 1)

    expect(stripLayout.height).toBeLessThanOrEqual(VIEWPORT.height)
    expect(stripLayout.widths[0]).toBe(Math.round((stripLayout.height * 612) / 792))
  })

  it('puts every page after the one before it, with a gap', () => {
    const stripLayout = row(book(4), VIEWPORT, 1)

    expect(stripLayout.starts[0]).toBe(GAP)
    for (let page = 1; page < 4; page++) {
      expect(stripLayout.starts[page]).toBe(stripLayout.starts[page - 1]! + stripLayout.widths[page - 1]! + GAP)
    }
    expect(stripLayout.length).toBe(stripLayout.starts[3]! + stripLayout.widths[3]! + GAP)
  })

  it('keeps each page its own shape', () => {
    // A book with a fold-out in it: one page twice the width of the rest, and
    // all of them the same height.
    const pages: Page[] = [
      { width: 612, height: 792 },
      { width: 1224, height: 792 },
      { width: 612, height: 792 },
    ]
    const stripLayout = row(pages, VIEWPORT, 1)

    // Within a pixel: each width is rounded on its own.
    expect(stripLayout.widths[1]).toBeCloseTo(stripLayout.widths[0]! * 2, -0.5)
    expect(stripLayout.widths[2]).toBe(stripLayout.widths[0])
  })

  it('lays out a page whose size nothing returned', () => {
    // A page the document could not measure stands here all the same, and the
    // pages after it are still where they were. A page at no width at all is a
    // row that jumps once its size is known.
    const stripLayout = row(
      [
        { width: 612, height: 792 },
        { width: 0, height: 0 },
        { width: 612, height: 792 },
      ],
      VIEWPORT,
      1,
    )

    expect(stripLayout.starts[1]).toBeGreaterThan(stripLayout.starts[0]!)
    expect(stripLayout.starts[2]).toBeGreaterThan(stripLayout.starts[1]!)
    expect(stripLayout.widths.every((wide) => wide > 0)).toBe(true)
  })

  describe('a page whose size is unusable', () => {
    const landscapePage: Page = { width: 1000, height: 500 }
    const height = Math.round(VIEWPORT.height - 2 * GAP)
    const getWidth = (page: Page) => Math.round((height * page.width) / page.height)
    const upright = getWidth({ width: 612, height: 792 })
    const broken: [string, Page][] = [
      ['no width', { width: 0, height: 100 }],
      ['no height', { width: 100, height: 0 }],
    ]

    it.each(broken)('takes the first page size when it has %s', (_, page) => {
      const stripLayout = row([landscapePage, page], VIEWPORT, 1)

      expect(stripLayout.widths).toEqual([getWidth(landscapePage), getWidth(landscapePage)])
    })

    it.each(broken)('takes an upright page when the first page has %s too', (_, page) => {
      const stripLayout = row([page, page], VIEWPORT, 1)

      expect(stripLayout.widths).toEqual([upright, upright])
    })
  })

  it('draws the pages larger when it is zoomed', () => {
    const one = row(book(4), VIEWPORT, 1)
    const two = row(book(4), VIEWPORT, 2)

    expect(two.height).toBeGreaterThan(one.height)
    expect(two.widths[0]).toBeGreaterThan(one.widths[0]!)
    expect(two.length).toBeGreaterThan(one.length)
  })
})

describe('which pages are drawn', () => {
  it('draws the pages in the viewport and a little either side', () => {
    const stripLayout = row(book(500), VIEWPORT, 1)

    const visiblePages = getPagesWithin(stripLayout, VIEWPORT, 0)

    expect(visiblePages[0]).toBe(0)
    expect(visiblePages.length).toBeLessThan(500)
    // Wide enough for the viewport, and for the reach beyond it on the far side.
    const isAcross = Math.ceil(VIEWPORT.width / stripLayout.widths[0]!)
    expect(visiblePages.length).toBeGreaterThan(isAcross)
  })

  it('draws around wherever the row has been scrolled to', () => {
    const stripLayout = row(book(500), VIEWPORT, 1)
    const at = 300

    const visiblePages = getPagesWithin(stripLayout, VIEWPORT, stripLayout.starts[at]!)

    expect(visiblePages).toContain(at)
    expect(visiblePages).not.toContain(0)
    expect(visiblePages.length).toBeLessThan(500)
  })

  it('pre-renders previous page for reverse navigation', () => {
    const stripLayout = row(book(500), VIEWPORT, 1)
    const at = 300

    const visiblePages = getPagesWithin(stripLayout, VIEWPORT, stripLayout.starts[at]!)

    expect(visiblePages).toContain(at - 1)
  })

  it('draws exactly the pages that touch the reach either side of the viewport', () => {
    // The reach is 1.5 viewports before the left edge and 1.5 after the right one.
    const viewport: Size = { width: 100, height: 100 }
    const stripLayout = {
      height: 50,
      starts: [0, 700, 1000, 1250, 1251],
      widths: [100, 150, 10, 10, 10],
      length: 1300,
    }

    // Page 1 ends on the near bound (850) and page 3 begins on the far one (1250).
    expect(getPagesWithin(stripLayout, viewport, 1000)).toEqual([1, 2, 3])
  })

  it('draws nothing for a document with no pages', () => {
    expect(getPagesWithin(row([], VIEWPORT, 1), VIEWPORT, 0)).toEqual([])
  })
})

describe('which page is in front', () => {
  it('is the one under the middle of the viewport', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    // A page scrolled halfway off is not the one being read: the middle of the
    // viewport is over the page after it.
    const half = stripLayout.starts[4]! - stripLayout.widths[4]! / 2

    expect(inFront(stripLayout, VIEWPORT, stripLayout.starts[4]!)).toBe(4)
    expect(inFront(stripLayout, { width: stripLayout.widths[0]!, height: VIEWPORT.height }, half)).toBe(4)
  })

  it('is the last page at the end of the row', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(inFront(stripLayout, VIEWPORT, stripLayout.length)).toBe(19)
  })

  it('is the first page at the beginning of the row', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(inFront(stripLayout, VIEWPORT, 0)).toBe(0)
  })
})

describe('where the row remains', () => {
  it('puts the page requested for against the left edge', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(findPageScroll(stripLayout, 5)).toBe(stripLayout.starts[5]! - GAP)
  })

  it('says nothing about a page the document does not have', () => {
    const stripLayout = row(book(4), VIEWPORT, 1)

    expect(findPageScroll(stripLayout, 9)).toBeUndefined()
  })

  it('holds the scroll inside the row, so the last page stops at the end of it', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(getScrollTarget(stripLayout, VIEWPORT, 0)).toBe(0)
    expect(getScrollTarget(stripLayout, VIEWPORT, 19)).toBe(stripLayout.length - VIEWPORT.width)
    expect(getScrollTarget(stripLayout, VIEWPORT, 5)).toBe(stripLayout.starts[5]! - GAP)
  })

  it('scrolls nowhere for a page the row does not hold', () => {
    expect(getScrollTarget(row(book(4), VIEWPORT, 1), VIEWPORT, 9)).toBeUndefined()
  })
  it('scrolls to the start of a row shorter than the viewport', () => {
    expect(
      getScrollTarget(
        row(book(1), { width: 4000, height: 800 }, 1),
        { width: 4000, height: 800 },
        0,
      ),
    ).toBe(0)
  })
})

describe('how close a page is drawn', () => {
  it('goes no further out than a whole page in the viewport', () => {
    // is room going to waste.
    expect(clampZoom(FURTHEST / NEARER)).toBe(FURTHEST)
    expect(clampZoom(FURTHEST)).toBeCloseTo(FURTHEST, 5)
  })

  it('goes no closer in than the closest', () => {
    expect(clampZoom(CLOSEST * NEARER)).toBe(CLOSEST)
  })

  it('leaves alone what remains between the two', () => {
    expect(clampZoom(2 * NEARER)).toBeCloseTo(2 * NEARER, 5)
    expect(clampZoom(2 / NEARER)).toBeCloseTo(2 / NEARER, 5)
  })
})

describe('a viewport nothing has been measured in', () => {
  it('makes no row at all', () => {
    // Until something has been measured there is no width to draw a page at,
    // and a page drawn at a made-up one is a page drawn and thrown away.
    const stripLayout = row(book(8), { width: 0, height: 0 }, 1)

    expect(stripLayout.height).toBe(0)
    expect(stripLayout.starts).toEqual([])
    expect(stripLayout.widths).toEqual([])
    expect(stripLayout.length).toBe(0)
    expect(getPagesWithin(stripLayout, { width: 0, height: 0 }, 0)).toEqual([])
  })

  it('makes no row in a viewport too short to stand a page in', () => {
    const stripLayout = row(book(8), { width: 900, height: 2 * GAP }, 1)

    expect(stripLayout.starts).toEqual([])
    expect(stripLayout.widths).toEqual([])
  })
})

describe('the page a key turns to', () => {
  it('turns on and back by one', () => {
    expect(getTurnTarget('ArrowRight', 2, 5)).toBe(3)
    expect(getTurnTarget('PageDown', 2, 5)).toBe(3)
    expect(getTurnTarget('ArrowLeft', 2, 5)).toBe(1)
  })

  it('goes to the ends under home and end', () => {
    expect(getTurnTarget('Home', 3, 5)).toBe(0)
    expect(getTurnTarget('End', 1, 5)).toBe(4)
  })

  it('turns nowhere past either end of the document', () => {
    expect(getTurnTarget('ArrowLeft', 0, 5)).toBeUndefined()
    expect(getTurnTarget('ArrowRight', 4, 5)).toBeUndefined()
  })

  it('turns nowhere for a key that turns no page, or a document with none', () => {
    expect(getTurnTarget('k', 1, 5)).toBeUndefined()
    expect(getTurnTarget('ArrowRight', 0, 0)).toBeUndefined()
  })
})
