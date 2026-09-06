import { describe, expect, it } from 'vitest'
import {
  CLOSEST,
  FURTHEST,
  GAP,
  NEARER,
  clamped,
  inFront,
  row,
  standAt,
  within,
  type Sheet,
  type Viewport,
} from './strip'

/** A viewport a page stands in whole: tall enough for a page, and a few wide. */
const VIEWPORT: Viewport = { wide: 900, high: 800 }

/** Every page the same shape, the way a book is. */
const book = (pages: number): Sheet[] =>
  Array.from({ length: pages }, () => ({ wide: 612, high: 792 }))

describe('the row a document makes', () => {
  it('draws a whole page in the viewport', () => {
    const laid = row(book(4), 4, VIEWPORT, 1)

    expect(laid.high).toBeLessThanOrEqual(VIEWPORT.high)
    expect(laid.widths[0]).toBe(Math.round((laid.high * 612) / 792))
  })

  it('puts every page after the one before it, with a gap', () => {
    const laid = row(book(4), 4, VIEWPORT, 1)

    expect(laid.starts[0]).toBe(GAP)
    for (let page = 1; page < 4; page++) {
      expect(laid.starts[page]).toBe(laid.starts[page - 1]! + laid.widths[page - 1]! + GAP)
    }
    expect(laid.length).toBe(laid.starts[3]! + laid.widths[3]! + GAP)
  })

  it('keeps each page its own shape', () => {
    // A book with a fold-out in it: one page twice the width of the rest, and
    // all of them the same height.
    const sheets: Sheet[] = [
      { wide: 612, high: 792 },
      { wide: 1224, high: 792 },
      { wide: 612, high: 792 },
    ]
    const laid = row(sheets, 3, VIEWPORT, 1)

    // Within a pixel: each width is rounded on its own.
    expect(laid.widths[1]).toBeCloseTo(laid.widths[0]! * 2, -0.5)
    expect(laid.widths[2]).toBe(laid.widths[0])
  })

  it('lays out a page whose size nothing said', () => {
    // A document answers with its pages' sizes, and one that has not answered
    // yet is still laid out. Every page at the same place is a row that jumps
    // as the sizes arrive.
    const laid = row([], 3, VIEWPORT, 1)

    expect(laid.starts[1]).toBeGreaterThan(laid.starts[0]!)
    expect(laid.widths.every((wide) => wide > 0)).toBe(true)
  })

  it('draws the pages larger when it is zoomed', () => {
    const one = row(book(4), 4, VIEWPORT, 1)
    const two = row(book(4), 4, VIEWPORT, 2)

    expect(two.high).toBeGreaterThan(one.high)
    expect(two.widths[0]).toBeGreaterThan(one.widths[0]!)
    expect(two.length).toBeGreaterThan(one.length)
  })
})

describe('which pages are drawn', () => {
  it('draws the pages in the viewport and a little either side', () => {
    const laid = row(book(500), 500, VIEWPORT, 1)

    const shown = within(laid, VIEWPORT, 0)

    expect(shown[0]).toBe(0)
    expect(shown.length).toBeLessThan(500)
    // Wide enough for the viewport, and for the reach beyond it on the far side.
    const across = Math.ceil(VIEWPORT.wide / laid.widths[0]!)
    expect(shown.length).toBeGreaterThan(across)
  })

  it('draws around wherever the row has been scrolled to', () => {
    const laid = row(book(500), 500, VIEWPORT, 1)
    const at = 300

    const shown = within(laid, VIEWPORT, laid.starts[at]!)

    expect(shown).toContain(at)
    expect(shown).not.toContain(0)
    expect(shown.length).toBeLessThan(500)
  })

  it('draws a page just left behind, in case the hand comes back', () => {
    const laid = row(book(500), 500, VIEWPORT, 1)
    const at = 300

    const shown = within(laid, VIEWPORT, laid.starts[at]!)

    expect(shown).toContain(at - 1)
  })

  it('draws nothing for a document with no pages', () => {
    expect(within(row([], 0, VIEWPORT, 1), VIEWPORT, 0)).toEqual([])
  })
})

describe('which page is in front', () => {
  it('is the one under the middle of the viewport', () => {
    const laid = row(book(20), 20, VIEWPORT, 1)

    // A page scrolled halfway off is not the one being read: the middle of the
    // viewport is over the page after it.
    const half = laid.starts[4]! - laid.widths[4]! / 2

    expect(inFront(laid, VIEWPORT, laid.starts[4]!)).toBe(4)
    expect(inFront(laid, { wide: laid.widths[0]!, high: VIEWPORT.high }, half)).toBe(4)
  })

  it('is the first page at the beginning of the row', () => {
    const laid = row(book(20), 20, VIEWPORT, 1)

    expect(inFront(laid, VIEWPORT, 0)).toBe(0)
  })
})

describe('where the row stands', () => {
  it('puts the page asked for against the left edge', () => {
    const laid = row(book(20), 20, VIEWPORT, 1)

    expect(standAt(laid, 5)).toBe(laid.starts[5]! - GAP)
  })

  it('says nothing about a page the document does not have', () => {
    const laid = row(book(4), 4, VIEWPORT, 1)

    expect(standAt(laid, 9)).toBeUndefined()
  })
})

describe('how close a page is drawn', () => {
  /** The ends a page is drawn between, which are what the controls carry. */
  const between = (zoom: number) => clamped(zoom, FURTHEST, CLOSEST)

  it('goes no further out than a whole page in the viewport', () => {
    // Further than that is a page smaller than the viewport it stands in, which
    // is room going to waste.
    expect(between(FURTHEST / NEARER)).toBe(FURTHEST)
    expect(between(FURTHEST)).toBeCloseTo(FURTHEST, 5)
  })

  it('goes no closer in than the closest', () => {
    expect(between(CLOSEST * NEARER)).toBe(CLOSEST)
  })

  it('leaves alone what stands between the two', () => {
    expect(between(2 * NEARER)).toBeCloseTo(2 * NEARER, 5)
    expect(between(2 / NEARER)).toBeCloseTo(2 / NEARER, 5)
  })
})

describe('a viewport nothing has been measured in', () => {
  it('makes no row at all', () => {
    // Until something has been measured there is no width to draw a page at,
    // and a page drawn at a made-up one is a page drawn and thrown away.
    const laid = row(book(8), 8, { wide: 0, high: 0 }, 1)

    expect(laid.high).toBe(0)
    expect(laid.widths).toEqual([])
    expect(laid.length).toBe(0)
    expect(within(laid, { wide: 0, high: 0 }, 0)).toEqual([])
  })

  it('makes no row in a viewport too short to stand a page in', () => {
    expect(row(book(8), 8, { wide: 900, high: 2 * GAP }, 1).widths).toEqual([])
  })
})
