import { describe, expect, it } from 'vitest'
import {
  CLOSEST,
  FURTHEST,
  GAP,
  NEARER,
  clampZoom,
  getPagesWithin,
  getCurrentPage,
  row,
  findPageScroll,
  getScrollTarget,
  getTurnTarget,
  getFitZoom,
  getAnchor,
  getAnchorScroll,
  getAcrossScroll,
  getAcrossShare,
  SHARP,
  type Page,
} from './strip'
import type { Size } from '@numen/ui'

const VIEWPORT: Size = { width: 900, height: 800 }

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
      expect(stripLayout.starts[page]).toBe(
        stripLayout.starts[page - 1]! + stripLayout.widths[page - 1]! + GAP,
      )
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
    // A page the document could not measure is still laid out, and the
    // pages after it keep their positions. A page with zero width would be a
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
    expect(stripLayout.widths.every((landscape) => landscape > 0)).toBe(true)
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
    const targetPage = 300

    const visiblePages = getPagesWithin(stripLayout, VIEWPORT, stripLayout.starts[targetPage]!)

    expect(visiblePages).toContain(targetPage)
    expect(visiblePages).not.toContain(0)
    expect(visiblePages.length).toBeLessThan(500)
  })

  it('pre-renders previous page for reverse navigation', () => {
    const stripLayout = row(book(500), VIEWPORT, 1)
    const targetPage = 300

    const visiblePages = getPagesWithin(stripLayout, VIEWPORT, stripLayout.starts[targetPage]!)

    expect(visiblePages).toContain(targetPage - 1)
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

    expect(getCurrentPage(stripLayout, VIEWPORT, stripLayout.starts[4]!)).toBe(4)
    expect(
      getCurrentPage(stripLayout, { width: stripLayout.widths[0]!, height: VIEWPORT.height }, half),
    ).toBe(4)
  })

  it('is the last page at the end of the row', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(getCurrentPage(stripLayout, VIEWPORT, stripLayout.length)).toBe(19)
  })

  it('is the first page at the beginning of the row', () => {
    const stripLayout = row(book(20), VIEWPORT, 1)

    expect(getCurrentPage(stripLayout, VIEWPORT, 0)).toBe(0)
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

  it('centers a page narrower than half the viewport', () => {
    const broadViewport: Size = { width: 1600, height: 800 }
    const stripLayout = row(book(20), broadViewport, 1)
    const target = getScrollTarget(stripLayout, broadViewport, 5)!

    const center = target + broadViewport.width / 2
    const pageCenter = stripLayout.starts[5]! + stripLayout.widths[5]! / 2
    expect(Math.abs(center - pageCenter)).toBeLessThanOrEqual(0.5)
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

describe('the current page after a scroll to a page', () => {
  const VIEWPORTS: readonly Size[] = [
    { width: 300, height: 800 },
    { width: 900, height: 800 },
    { width: 1250, height: 800 },
    { width: 1600, height: 800 },
    { width: 2600, height: 700 },
  ]

  it('is the page scrolled to, in every viewport width', () => {
    for (const viewport of VIEWPORTS) {
      const stripLayout = row(book(40), viewport, 1)
      const furthest = stripLayout.length - viewport.width
      for (let page = 0; page < 40; page++) {
        const target = getScrollTarget(stripLayout, viewport, page)!
        if (target === 0 || target === furthest) continue
        expect({ viewport, page, current: getCurrentPage(stripLayout, viewport, target) }).toEqual({
          viewport,
          page,
          current: page,
        })
      }
    }
  })

  it('stays the same page when the viewport height changes and the row is scrolled to it again', () => {
    const before: Size = { width: 1400, height: 800 }
    const after: Size = { width: 1400, height: 760 }
    const page = getCurrentPage(
      row(book(40), before, 1),
      before,
      getScrollTarget(row(book(40), before, 1), before, 12)!,
    )
    const afterLayout = row(book(40), after, 1)

    expect(getCurrentPage(afterLayout, after, getScrollTarget(afterLayout, after, page)!)).toBe(
      page,
    )
  })

  it('is one page on either side of a border, and changes exactly at it', () => {
    const stripLayout = row(book(40), VIEWPORT, 1)
    const border = stripLayout.starts[12]! - VIEWPORT.width / 2

    expect(getCurrentPage(stripLayout, VIEWPORT, border - 1)).toBe(11)
    expect(getCurrentPage(stripLayout, VIEWPORT, border - 0.5)).toBe(11)
    expect(getCurrentPage(stripLayout, VIEWPORT, border)).toBe(12)
    expect(getCurrentPage(stripLayout, VIEWPORT, border + 0.5)).toBe(12)
  })
})

describe('how close a page is drawn', () => {
  it('goes no further out than a whole page in the viewport', () => {
    // Zooming out beyond one whole page in the viewport wastes space.
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

describe('the zoom a document opens at', () => {
  it('draws a whole page in the viewport for the page fit', () => {
    expect(getFitZoom('page', book(4), VIEWPORT)).toBe(FURTHEST)
  })

  it('draws the first page as landscape as the viewport for the width fit', () => {
    const landscape: Size = { width: 1600, height: 800 }
    const zoom = getFitZoom('width', book(4), landscape)

    expect(row(book(4), landscape, zoom).widths[0]).toBeCloseTo(landscape.width - 2 * GAP, -1)
  })

  it('draws the first page at its printed size for the 100% fit', () => {
    const zoom = getFitZoom('100%', book(4), VIEWPORT)

    expect(row(book(4), VIEWPORT, zoom).height).toBe(Math.round((792 * 96) / 72))
  })

  it('draws the page smaller than a whole page in a viewport narrower than it', () => {
    const narrow: Size = { width: 300, height: 800 }
    const zoom = getFitZoom('width', book(4), narrow)

    expect(zoom).toBeLessThan(FURTHEST)
    expect(row(book(4), narrow, zoom).widths[0]).toBeCloseTo(narrow.width - 2 * GAP, -1)
  })

  it('never draws the page closer than the closest zoom', () => {
    expect(getFitZoom('width', book(4), { width: 9000, height: 300 })).toBe(CLOSEST)
  })

  it('draws a whole page in a viewport with no height', () => {
    expect(getFitZoom('width', book(4), { width: 900, height: 0 })).toBe(FURTHEST)
  })
})

describe('the point kept under the center line of the viewport', () => {
  const viewport = { width: 400, height: 300 }
  const pages = [
    { width: 100, height: 200 },
    { width: 100, height: 200 },
  ]

  it('is a page and a share of its width', () => {
    const before = row(pages, viewport, 1)
    const center = before.starts[1]! + before.widths[1]! / 4

    const anchor = getAnchor(before, viewport, center - viewport.width / 2)

    expect(anchor?.page).toBe(1)
    expect(anchor?.fraction).toBeCloseTo(0.25)
  })

  it('is nothing in a row with no pages', () => {
    expect(getAnchor(row([], viewport, 1), viewport, 0)).toBeUndefined()
  })

  it('stays under the center line when every page is wider', () => {
    const before = row(pages, viewport, 1)
    const after = row(pages, viewport, 2)
    const offset = before.starts[1]! + before.widths[1]! / 4 - viewport.width / 2
    const anchor = getAnchor(before, viewport, offset)!

    const to = getAnchorScroll(after, viewport, anchor)!

    const center = to + viewport.width / 2
    expect((center - after.starts[1]!) / after.widths[1]!).toBeCloseTo(0.25, 1)
  })

  it('is clamped to the row', () => {
    const after = row(pages, viewport, 1)

    expect(getAnchorScroll(after, viewport, { page: 1, fraction: 5 })).toBe(
      Math.max(after.length - viewport.width, 0),
    )
    expect(getAnchorScroll(after, viewport, { page: 0, fraction: -5 })).toBe(0)
    expect(getAnchorScroll(after, viewport, { page: 9, fraction: 0 })).toBeUndefined()
  })

  it('keeps the share of the page height under the center line across', () => {
    const before = row(pages, viewport, 1)
    const after = row(pages, viewport, 2)
    const share = getAcrossShare(
      before,
      viewport,
      Math.max(Math.round(GAP + 0.5 * before.height - viewport.height / 2), 0),
    )

    const to = getAcrossScroll(after, viewport, share)

    expect((to + viewport.height / 2 - GAP) / after.height).toBeCloseTo(0.5, 1)
    expect(getAcrossShare(row(pages, { width: 400, height: 0 }, 1), viewport, 0)).toBe(0.5)
  })
})

describe('the pages requested at the settled width', () => {
  it('are fewer than the pages drawn, with a smaller margin', () => {
    const viewport = { width: 400, height: 300 }
    const long = row(
      Array.from({ length: 40 }, () => ({ width: 100, height: 200 })),
      viewport,
      1,
    )

    const allPages = getPagesWithin(long, viewport, 2000)
    const sharp = getPagesWithin(long, viewport, 2000, SHARP)

    expect(sharp.length).toBeLessThan(allPages.length)
    expect(sharp.every((page) => allPages.includes(page))).toBe(true)
    expect(sharp.length).toBeGreaterThan(0)
  })
})
