import { describe, expect, it } from 'vitest'
import {
  GAP,
  getColumn,
  NARROWEST,
  columnHeight,
  columnWidth,
  countColumns,
  countFilledColumns,
  columnsInAll,
  findSpreadAt,
  getSpreadStart,
  inFront,
  leftInDocument,
  createPages,
  getSpread,
  countSpreads,
  type Flow,
  type Mark,
} from './spread'
import { countBytes, getUnitOffset } from './bytes'
import { BOOK_WORDS } from './words'
import { EDGE, SWIPE, goToPage, dragPage, pressPage, swipePage } from './pageStep'
import type { Span } from '@numen/ui'
import type { PageNavigationAction } from '@numen/ui'

/** A wide reading area, which takes two columns, and a narrow one, which takes one. */
const WIDE = 800
const NARROW = 420

/**
 * The columns a document of so many of them evaluates to, in a given reading area.
 * A column takes its share of the area, gap and all, which is how the browser
 * lays them out and how far a spread has to move.
 */
const createFlow = (width: number, columns: number, all: number): Flow => ({
  contentWidth: (all * width) / columns,
  width,
  gap: GAP,
  columns,
})

describe('countColumns', () => {
  it('returns two columns when available width fits minimum column width', () => {
    expect(countColumns(2 * (NARROWEST + GAP), 1)).toBe(2)
    expect(countColumns(2 * (NARROWEST + GAP) - 1, 1)).toBe(1)
  })

  it('returns one column when width is zero', () => {
    expect(countColumns(0, 1)).toBe(1)
  })

  it('accounts for text scale factor', () => {
    const availableWidth = 2 * (NARROWEST + GAP)
    expect(countColumns(availableWidth, 1)).toBe(2)
    expect(countColumns(availableWidth, 1.5)).toBe(1)
  })

  it('calculates column width subtracting gap', () => {
    expect(columnWidth({ contentWidth: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(WIDE / 2 - GAP)
    expect(columnWidth({ contentWidth: 0, width: NARROW, gap: GAP, columns: 1 })).toBe(NARROW - GAP)
  })
})

describe('columnHeight', () => {
  it('fits whole lines within available height', () => {
    expect(columnHeight(1000, 24)).toBe(984)
    expect(columnHeight(984, 24)).toBe(984)
  })

  it('returns minimum one line height when available height is too short', () => {
    expect(columnHeight(20, 24)).toBe(24)
  })

  it('returns total height when line height is zero', () => {
    expect(columnHeight(1000, 0)).toBe(1000)
  })

  it('returns zero when area height is zero', () => {
    expect(columnHeight(0, 24)).toBe(0)
  })
})

describe('countFilledColumns', () => {
  const flow = { contentWidth: 2 * NARROW + GAP, width: NARROW, gap: GAP, columns: 1 }
  const second = NARROW + GAP

  it('determines filled columns from the furthest mark position', () => {
    expect(countFilledColumns([{ at: 0, x: 0 }], flow)).toBe(1)
    expect(
      countFilledColumns(
        [
          { at: 0, x: 0 },
          { at: 40, x: second },
        ],
        flow,
      ),
    ).toBe(2)
  })

  it('includes marks within rounding tolerance into the corresponding column', () => {
    expect(
      countFilledColumns(
        [
          { at: 0, x: 0 },
          { at: 40, x: second - 0.1875 },
        ],
        flow,
      ),
    ).toBe(2)
    expect(
      countFilledColumns(
        [
          { at: 0, x: 0 },
          { at: 40, x: second + 0.1875 },
        ],
        flow,
      ),
    ).toBe(2)
  })

  it('includes mark placed at column boundary', () => {
    expect(countFilledColumns([{ at: 0, x: NARROW - 1 }], flow)).toBe(1)
  })

  it('returns zero when marks list is empty or width is zero', () => {
    expect(countFilledColumns([], flow)).toBe(0)
    expect(countFilledColumns([{ at: 0, x: 0 }], { ...flow, width: 0 })).toBe(0)
  })
})

describe('countSpreads and columnsInAll', () => {
  it('calculates total column count from content width', () => {
    expect(columnsInAll(createFlow(WIDE, 2, 10))).toBe(10)
    expect(columnsInAll(createFlow(NARROW, 1, 7))).toBe(7)
  })

  it('converts columns to spreads based on columns per spread', () => {
    expect(countSpreads(createFlow(WIDE, 2, 10))).toBe(5)
    expect(countSpreads(createFlow(WIDE, 2, 9))).toBe(5)
    expect(countSpreads(createFlow(NARROW, 1, 7))).toBe(7)
  })

  it('does not add extra spread for trailing gaps', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(countSpreads({ ...flow, contentWidth: flow.contentWidth + GAP })).toBe(5)
  })

  it('does not add extra spread for fractional pixel rounding', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(countSpreads({ ...flow, contentWidth: flow.contentWidth + 0.5 })).toBe(5)
    expect(countSpreads({ ...flow, contentWidth: flow.contentWidth - 0.5 })).toBe(5)
  })

  it('returns zero for empty content', () => {
    expect(countSpreads({ contentWidth: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(0)
    expect(columnsInAll({ contentWidth: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(0)
  })

  it('returns one spread for single-line content', () => {
    expect(countSpreads(createFlow(WIDE, 2, 1))).toBe(1)
    expect(countSpreads(createFlow(NARROW, 1, 1))).toBe(1)
  })

  it('returns zero when container width is zero', () => {
    expect(countSpreads({ contentWidth: 1000, width: 0, gap: GAP, columns: 1 })).toBe(0)
  })
})

describe('getSpreadStart', () => {
  it('computes spread offset as a multiple of column pairs and gaps', () => {
    const flow = createFlow(WIDE, 2, 200)
    const one = columnWidth(flow)
    expect(getSpreadStart(flow, 100)).toBe(200 * (one + GAP))
  })

  it('advances starting offset by one container width per spread', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(getSpreadStart(flow, 1) - getSpreadStart(flow, 0)).toBe(WIDE)
  })
})

describe('getSpread', () => {
  it('maps horizontal position to the correct spread index', () => {
    const flow = createFlow(WIDE, 2, 10)
    const step = columnWidth(flow) + GAP

    expect(getSpread(flow, 0)).toBe(0)
    expect(getSpread(flow, step)).toBe(0)
    expect(getSpread(flow, 2 * step)).toBe(1)
    expect(getSpread(flow, 3 * step)).toBe(1)
    expect(getSpread(flow, 8 * step)).toBe(4)
  })

  it('clamps spread index to the valid range', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(getSpread(flow, 100_000)).toBe(4)
    expect(getSpread(flow, -100)).toBe(0)
  })
})

describe('inFront', () => {
  const createRuns = (flow: Flow, count: number): Mark[] => {
    const step = columnWidth(flow) + GAP
    return Array.from({ length: count }, (_, index) => ({ at: index * 100, x: index * step }))
  }

  it('returns byte offset of first mark in spread', () => {
    const flow = createFlow(WIDE, 2, 10)
    const marks = createRuns(flow, 10)

    expect(inFront(marks, flow, 0)).toBe(0)
    expect(inFront(marks, flow, 1)).toBe(200)
    expect(inFront(marks, flow, 4)).toBe(800)
  })

  it('returns undefined when spread contains no marks', () => {
    const flow = createFlow(WIDE, 2, 10)
    const marks: Mark[] = [
      { at: 0, x: 0 },
      { at: 500, x: getSpreadStart(flow, 3) },
    ]

    expect(inFront(marks, flow, 2)).toBeUndefined()
  })

  it('returns undefined when marks list is empty', () => {
    expect(inFront([], createFlow(WIDE, 2, 1), 0)).toBeUndefined()
  })
})

describe('findSpreadAt', () => {
  const flow = createFlow(WIDE, 2, 10)
  const step = columnWidth(flow) + GAP
  const marks: Mark[] = [
    { at: 0, x: 0 },
    { at: 100, x: 3 * step },
    { at: 240, x: 7 * step },
  ]

  it('finds spread index containing or preceding the given byte offset', () => {
    expect(findSpreadAt(marks, flow, 0)).toBe(0)
    expect(findSpreadAt(marks, flow, 99)).toBe(0)
    expect(findSpreadAt(marks, flow, 100)).toBe(1)
    expect(findSpreadAt(marks, flow, 180)).toBe(1)
    expect(findSpreadAt(marks, flow, 240)).toBe(3)
  })

  it('returns spread zero for offsets before the first mark', () => {
    expect(findSpreadAt(marks, flow, -5)).toBe(0)
    expect(findSpreadAt([], flow, 900)).toBe(0)
  })

  it('returns final spread for offsets beyond all marks', () => {
    expect(findSpreadAt(marks, flow, 10_000)).toBe(3)
  })
})

describe('countBytes and getUnitOffset', () => {
  it('counts ASCII character as one byte', () => {
    expect(countBytes('abc')).toBe(3)
    expect(getUnitOffset('abcdef', 3)).toBe(3)
  })

  it('counts multi-byte UTF-8 sequences', () => {
    expect(countBytes('да')).toBe(4)
    expect(countBytes('सत्य')).toBe(12)

    expect(getUnitOffset('дальше', 4)).toBe(2)
    expect(getUnitOffset('सत्यम्', 6)).toBe(2)
  })

  it('aligns to character boundaries when byte offset falls inside a character', () => {
    expect(getUnitOffset('सत्य', 4)).toBe(1)
    expect(getUnitOffset('सत्य', 5)).toBe(1)
    expect(getUnitOffset('सत्य', 6)).toBe(2)
  })

  it('counts surrogate pairs as two UTF-16 code units', () => {
    expect(countBytes('𑀓')).toBe(4)
    expect(getUnitOffset('𑀓a', 4)).toBe(2)
    expect(getUnitOffset('𑀓a', 5)).toBe(3)
  })

  it('returns zero for non-positive byte offsets', () => {
    expect(getUnitOffset('सत्य', 0)).toBe(0)
    expect(getUnitOffset('सत्य', -3)).toBe(0)
    expect(getUnitOffset('', 5)).toBe(0)
  })

  it('clamps to string length when byte offset exceeds content length', () => {
    expect(getUnitOffset('да', 99)).toBe(2)
  })
})

describe('swipePage and pressPage', () => {
  it('maps swipe leftward to next page and swipe rightward to previous page', () => {
    expect(swipePage(-SWIPE)).toBe('next')
    expect(swipePage(SWIPE)).toBe('back')
  })

  it('ignores swipe when displacement is below threshold', () => {
    expect(swipePage(0)).toBeUndefined()
    expect(swipePage(SWIPE - 1)).toBeUndefined()
    expect(swipePage(1 - SWIPE)).toBeUndefined()
  })

  it('maps edge presses to page turns', () => {
    expect(pressPage(0, WIDE)).toBe('back')
    expect(pressPage(WIDE * EDGE, WIDE)).toBe('back')
    expect(pressPage(WIDE, WIDE)).toBe('next')
    expect(pressPage(WIDE * (1 - EDGE), WIDE)).toBe('next')
  })

  it('ignores presses in the central area', () => {
    expect(pressPage(WIDE / 2, WIDE)).toBeUndefined()
    expect(pressPage(10, 0)).toBeUndefined()
  })
})

describe('dragPage', () => {
  const turn = (from: number, to: number, hasSelection = false) =>
    dragPage(from, to, { wide: WIDE, hasSelection })

  it('determines page turn direction from swipe gesture', () => {
    expect(turn(500, 500 - SWIPE)).toBe('next')
    expect(turn(500, 500 + SWIPE)).toBe('back')
  })

  it('triggers page turn on static tap near edges', () => {
    expect(turn(10, 10)).toBe('back')
    expect(turn(WIDE - 10, WIDE - 10)).toBe('next')
  })

  it('ignores gestures when text selection is active', () => {
    expect(turn(500, 500 - SWIPE, true)).toBeUndefined()
    expect(turn(WIDE - 10, WIDE - 10, true)).toBeUndefined()
    expect(turn(10, 10, true)).toBeUndefined()
  })
})

describe('createPages', () => {
  const flow = { contentWidth: 10 * WIDE, width: WIDE, gap: GAP, columns: 2 }
  const document = { from: 2000, to: 3000 }
  const book = { from: 0, to: 10_000 }

  const createMarks = (columns: number): Mark[] =>
    Array.from({ length: columns }, (_, column) => ({
      at: document.from + column,
      x: column * (columnWidth(flow) + GAP),
    }))

  const marks = createMarks(columnsInAll(flow))
  const here = marks.length
  const perColumn = (document.to - document.from) / here
  const before = Math.round((document.from - book.from) / perColumn)
  const pages = (b: Span, d: Span, spread: number, m: readonly Mark[] = marks) =>
    createPages({ book: b, document: d, flow, spread, marks: m })

  it('advances page count by columns rendered per spread', () => {
    const first = pages(book, document, 0)
    const next = pages(book, document, 1)
    expect(next.page - first.page).toBe(2)
  })

  it('calculates page number based on byte distribution', () => {
    expect(pages(book, document, 0).page).toBe(before + 1)
    expect(pages(book, document, 0).pages).toBe(Math.round((book.to - book.from) / perColumn))
  })

  it('counts only columns with actual content', () => {
    const one = createMarks(1)
    expect(countFilledColumns(one, flow)).toBe(1)
    expect(pages({ ...document }, document, 0, one)).toEqual({ page: 1, pages: 1 })
  })

  it('defaults to page 1 of 1 when no marks exist', () => {
    expect(pages(book, document, 0, [])).toEqual({ page: 1, pages: 1 })
  })
})

describe('leftInDocument', () => {
  const flow: Flow = { contentWidth: 2700, width: 800, gap: GAP, columns: 2 }
  const marks: readonly Mark[] = [
    { at: 0, x: 10 },
    { at: 400, x: 1610 },
  ]

  it('returns count of columns remaining after active spread', () => {
    expect(leftInDocument(flow, 0, marks)).toBe(3)
  })

  it('returns zero when active spread is the final spread', () => {
    expect(leftInDocument(flow, 2, marks)).toBe(0)
  })
})

describe('BOOK_WORDS', () => {
  it('formats active page against total pages', () => {
    expect(BOOK_WORDS.of(3, 40)).toBe('3 of 40')
  })

  it('formats remaining pages count in singular and plural forms', () => {
    expect(BOOK_WORDS.left(1)).toBe('1 page left in chapter')
    expect(BOOK_WORDS.left(5)).toBe('5 pages left in chapter')
  })
})

describe('goToPage', () => {
  const span = { from: 400, to: 900 }
  const book = { from: 0, to: 1400 }
  const to = (way: PageNavigationAction, spread: number, count: number, s = span, b = book) =>
    goToPage(way, { spread, count, span: s, book: b })

  it('navigates to target spread within document boundaries', () => {
    expect(to('next', 0, 3)).toStrictEqual({ spread: 1 })
    expect(to('back', 2, 3)).toStrictEqual({ spread: 1 })
    expect(to('first', 1, 3)).toStrictEqual({ spread: 0 })
    expect(to('first', 2, 3)).toStrictEqual({ spread: 0 })
    expect(to('last', 1, 3)).toStrictEqual({ spread: 2 })
  })

  it('transitions to adjacent document offsets at boundary edges', () => {
    expect(to('next', 2, 3)).toStrictEqual({ offset: 900 })
    expect(to('back', 0, 3)).toStrictEqual({ offset: 399 })
    expect(to('back', 1, 3)).toStrictEqual({ spread: 0 })
    expect(to('back', 0, 3, span, { from: 399, to: 1400 })).toStrictEqual({ offset: 399 })
  })

  it('returns empty result at start or end of book', () => {
    expect(to('next', 2, 3, span, span)).toStrictEqual({})
    expect(to('back', 0, 3, span, span)).toStrictEqual({})
  })
})

describe('zero columns layout', () => {
  it('returns zero column width when columns count is zero', () => {
    expect(columnWidth({ contentWidth: 500, width: 800, gap: GAP, columns: 0 })).toBe(0)
  })

  it('evaluates to zero columns and zero spreads when dimensions are zero', () => {
    expect(columnsInAll({ contentWidth: 500, width: 0, gap: GAP, columns: 2 })).toBe(0)
    expect(countSpreads({ contentWidth: 500, width: 800, gap: GAP, columns: 0 })).toBe(0)
  })
})

describe('spread index when gap fills column', () => {
  it('returns spread zero when column width is zero', () => {
    const flow = { contentWidth: 500, width: 100, gap: GAP, columns: 2 }
    expect(columnWidth(flow)).toBe(0)
    expect(getSpread(flow, 300)).toBe(0)
  })
})

describe('createPages with byte distribution', () => {
  const flow = { contentWidth: 10 * WIDE, width: WIDE, gap: GAP, columns: 2 }
  const marks: Mark[] = [
    { at: 100, x: 0 },
    { at: 200, x: columnWidth(flow) + GAP },
  ]

  it('calculates page count when document has zero bytes', () => {
    const empty = { from: 100, to: 100 }
    expect(
      createPages({ book: { from: 100, to: 5000 }, document: empty, flow, spread: 0, marks }),
    ).toStrictEqual({
      page: 1,
      pages: 2,
    })
  })

  it('clamps active page index to minimum 1', () => {
    const empty = { from: 100, to: 100 }
    expect(
      createPages({ book: { from: 0, to: 1000 }, document: empty, flow, spread: -1, marks }).page,
    ).toBe(1)
  })

  it('ensures total pages covers document end position', () => {
    expect(
      createPages({
        book: { from: 0, to: 550 },
        document: { from: 500, to: 1000 },
        flow,
        spread: 0,
        marks,
      }),
    ).toStrictEqual({
      page: 3,
      pages: 4,
    })
  })
})

describe('getColumn with unmeasured dimensions', () => {
  it('returns column index zero', () => {
    const flow = { contentWidth: 500, width: 0, gap: GAP, columns: 2 }
    expect(getColumn(flow, 0)).toBe(0)
    expect(getColumn(flow, 300)).toBe(0)
  })
})
