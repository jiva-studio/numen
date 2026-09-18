import { describe, expect, it } from 'vitest'
import {
  GAP,
  NARROWEST,
  columnHeight,
  columnWidth,
  columnsIn,
  countFilledColumns,
  columnsInAll,
  findSpreadAt,
  getSpreadStart,
  inFront,
  leftInDocument,
  pagesOf,
  spreadAt,
  countSpreads,
  type Flow,
  type Mark,
} from './spread'
import { bytesIn, unitsIn } from './bytes'
import { BOOK_WORDS } from './words'
import { EDGE, SWIPE, handTurn, pressTurn, swipeTurn, turnTo } from './turn'

/** A wide reading area, which takes two columns, and a narrow one, which takes one. */
const WIDE = 800
const NARROW = 420

/**
 * The columns a document of so many of them comes to, in a given reading area.
 * A column takes its share of the area, gap and all, which is how the browser
 * lays them out and how far a spread has to move.
 */
const createFlow = (width: number, columns: number, all: number): Flow => ({
  along: (all * width) / columns,
  width,
  gap: GAP,
  columns,
})

describe('how many columns a reading area takes', () => {
  it('takes two once each of them can be set at the narrowest a column is read at', () => {
    // Each column keeps a gap of its own, at the edge of the area as between
    // the two of them.
    expect(columnsIn(2 * (NARROWEST + GAP), 1)).toBe(2)
    expect(columnsIn(2 * (NARROWEST + GAP) - 1, 1)).toBe(1)
  })

  it('takes one in an area nothing has been measured in', () => {
    expect(columnsIn(0, 1)).toBe(1)
  })

  it('takes the size the text is set at into account', () => {
    // A column is measured in characters, so text set half again as large needs
    // half again as much room before a second column will hold a line.
    const wide = 2 * (NARROWEST + GAP)
    expect(columnsIn(wide, 1)).toBe(2)
    expect(columnsIn(wide, 1.5)).toBe(1)
  })

  it('sets each column to its share of the area, less the gap it keeps', () => {
    expect(columnWidth({ along: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(WIDE / 2 - GAP)
    expect(columnWidth({ along: 0, width: NARROW, gap: GAP, columns: 1 })).toBe(NARROW - GAP)
  })
})

describe('how tall a column is set', () => {
  it('is the lines that fit whole in the area, and no part of another', () => {
    expect(columnHeight(1000, 24)).toBe(984)
    expect(columnHeight(984, 24)).toBe(984)
  })

  it('is one line in an area too short to hold one', () => {
    // The line runs past the foot of the area either way, and a column of no
    // height at all holds nothing and never ends.
    expect(columnHeight(20, 24)).toBe(24)
  })

  it('is the whole of the area where the lines are of no known height', () => {
    expect(columnHeight(1000, 0)).toBe(1000)
  })

  it('is nothing in an area nothing has been measured in', () => {
    expect(columnHeight(0, 24)).toBe(0)
  })
})

describe('how many columns the text of a document fills', () => {
  /** One column of a narrow area, and the place the second one begins. */
  const flow = { along: 2 * NARROW + GAP, width: NARROW, gap: GAP, columns: 1 }
  const second = NARROW + GAP

  it('is counted off the run standing getFurthest along them', () => {
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

  it('counts a run measured a fraction of a pixel short of its column into it', () => {
    // The browser lays the columns out in whole device pixels, so the run at
    // the head of the second column was measured at 654.8125 where the column
    // was reckoned to begin at 655, and the whole of it went uncounted.
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

  it('counts a run standing at the foot of a column into that column', () => {
    expect(countFilledColumns([{ at: 0, x: NARROW - 1 }], flow)).toBe(1)
  })

  it('is no column at all where nothing was laid out', () => {
    expect(countFilledColumns([], flow)).toBe(0)
    expect(countFilledColumns([{ at: 0, x: 0 }], { ...flow, width: 0 })).toBe(0)
  })
})

describe('how many countSpreads a document comes to', () => {
  it('counts the columns off the whole run', () => {
    expect(columnsInAll(createFlow(WIDE, 2, 10))).toBe(10)
    expect(columnsInAll(createFlow(NARROW, 1, 7))).toBe(7)
  })

  it('turns them into countSpreads, two columns to a spread in a wide area', () => {
    expect(countSpreads(createFlow(WIDE, 2, 10))).toBe(5)
    expect(countSpreads(createFlow(WIDE, 2, 9))).toBe(5)
    expect(countSpreads(createFlow(NARROW, 1, 7))).toBe(7)
  })

  it('adds no spread for a gap left standing after the last column', () => {
    // A run measured one gap longer than the columns in it is the same
    // document, and a spread of nothing at the end of a book is a page a person
    // turns to and finds empty.
    const flow = createFlow(WIDE, 2, 10)
    expect(countSpreads({ ...flow, along: flow.along + GAP })).toBe(5)
  })

  it('adds no spread for the pixel a browser rounds a column to', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(countSpreads({ ...flow, along: flow.along + 0.5 })).toBe(5)
    expect(countSpreads({ ...flow, along: flow.along - 0.5 })).toBe(5)
  })

  it('comes to nothing for a document with no text at all', () => {
    expect(countSpreads({ along: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(0)
    expect(columnsInAll({ along: 0, width: WIDE, gap: GAP, columns: 2 })).toBe(0)
  })

  it('comes to one spread for a document of one line', () => {
    expect(countSpreads(createFlow(WIDE, 2, 1))).toBe(1)
    expect(countSpreads(createFlow(NARROW, 1, 1))).toBe(1)
  })

  it('comes to nothing in an area nothing has been measured in', () => {
    expect(countSpreads({ along: 1000, width: 0, gap: GAP, columns: 1 })).toBe(0)
  })
})

describe('where a spread begins', () => {
  it('stands a whole number of countSpreads out from the first', () => {
    const flow = createFlow(WIDE, 2, 200)

    // A hundred countSpreads out, the place is still exactly the place the hundredth
    // column pair begins at: nothing is added up along the way.
    const one = columnWidth(flow)
    expect(getSpreadStart(flow, 100)).toBe(200 * (one + GAP))
  })

  it('begins one whole reading area along from the spread before it', () => {
    // The gap a column keeps stands inside the area at either edge, so nothing
    // stands between two countSpreads for the turn to carry over.
    const flow = createFlow(WIDE, 2, 10)
    expect(getSpreadStart(flow, 1) - getSpreadStart(flow, 0)).toBe(WIDE)
  })
})

describe('which spread a place falls in', () => {
  it('is the spread the column at that place belongs to', () => {
    const flow = createFlow(WIDE, 2, 10)
    const step = columnWidth(flow) + GAP

    expect(spreadAt(flow, 0)).toBe(0)
    expect(spreadAt(flow, step)).toBe(0)
    expect(spreadAt(flow, 2 * step)).toBe(1)
    expect(spreadAt(flow, 3 * step)).toBe(1)
    expect(spreadAt(flow, 8 * step)).toBe(4)
  })

  it('goes no further than the last spread there is', () => {
    const flow = createFlow(WIDE, 2, 10)
    expect(spreadAt(flow, 100_000)).toBe(4)
    expect(spreadAt(flow, -100)).toBe(0)
  })
})

describe('which run of the text is in front', () => {
  /** Runs an offset apart, each one column further along. */
  const createRuns = (flow: Flow, count: number): Mark[] => {
    const step = columnWidth(flow) + GAP
    return Array.from({ length: count }, (_, index) => ({ at: index * 100, x: index * step }))
  }

  it('is the first run standing in the spread', () => {
    const flow = createFlow(WIDE, 2, 10)
    const marks = createRuns(flow, 10)

    expect(inFront(marks, flow, 0)).toBe(0)
    expect(inFront(marks, flow, 1)).toBe(200)
    expect(inFront(marks, flow, 4)).toBe(800)
  })

  it('is nothing where no run stands in the spread', () => {
    const flow = createFlow(WIDE, 2, 10)
    // A picture filling the third spread on its own, with no run of text in it.
    const marks: Mark[] = [
      { at: 0, x: 0 },
      { at: 500, x: getSpreadStart(flow, 3) },
    ]

    expect(inFront(marks, flow, 2)).toBeUndefined()
  })

  it('is nothing at all where the document has no runs', () => {
    expect(inFront([], createFlow(WIDE, 2, 1), 0)).toBeUndefined()
  })
})

describe('which spread an offset stands in', () => {
  const flow = createFlow(WIDE, 2, 10)
  const step = columnWidth(flow) + GAP
  const marks: Mark[] = [
    { at: 0, x: 0 },
    { at: 100, x: 3 * step },
    { at: 240, x: 7 * step },
  ]

  it('is the spread of the last run beginning at or before it', () => {
    expect(findSpreadAt(marks, flow, 0)).toBe(0)
    expect(findSpreadAt(marks, flow, 99)).toBe(0)
    expect(findSpreadAt(marks, flow, 100)).toBe(1)
    expect(findSpreadAt(marks, flow, 180)).toBe(1)
    expect(findSpreadAt(marks, flow, 240)).toBe(3)
  })

  it('is the first spread for an offset before anything the document carries', () => {
    expect(findSpreadAt(marks, flow, -5)).toBe(0)
    expect(findSpreadAt([], flow, 900)).toBe(0)
  })

  it('is the spread of the last run for an offset past everything', () => {
    expect(findSpreadAt(marks, flow, 10_000)).toBe(3)
  })
})

describe('a byte offset read into a string', () => {
  it('counts a Latin character as one byte', () => {
    expect(bytesIn('abc')).toBe(3)
    expect(unitsIn('abcdef', 3)).toBe(3)
  })

  it('counts a Cyrillic character as two, and a Devanagari one as three', () => {
    expect(bytesIn('да')).toBe(4)
    expect(bytesIn('सत्य')).toBe(12)

    // Four bytes into the Cyrillic is two characters, not four.
    expect(unitsIn('дальше', 4)).toBe(2)
    expect(unitsIn('सत्यम्', 6)).toBe(2)
  })

  it('stops short of a character the bytes do not reach the end of', () => {
    // Half a Devanagari character is not a place in a string.
    expect(unitsIn('सत्य', 4)).toBe(1)
    expect(unitsIn('सत्य', 5)).toBe(1)
    expect(unitsIn('सत्य', 6)).toBe(2)
  })

  it('counts a character written in two units as two', () => {
    // Four bytes of UTF-8, two units of UTF-16.
    expect(bytesIn('𑀓')).toBe(4)
    expect(unitsIn('𑀓a', 4)).toBe(2)
    expect(unitsIn('𑀓a', 5)).toBe(3)
  })

  it('reaches the start of a string for no bytes at all', () => {
    expect(unitsIn('सत्य', 0)).toBe(0)
    expect(unitsIn('सत्य', -3)).toBe(0)
    expect(unitsIn('', 5)).toBe(0)
  })

  it('reaches the end of a string for more bytes than it holds', () => {
    expect(unitsIn('да', 99)).toBe(2)
  })
})

describe('what turns the page', () => {
  it('follows the hand: a swipe leftward brings the page after this one', () => {
    expect(swipeTurn(-SWIPE)).toBe('next')
    expect(swipeTurn(SWIPE)).toBe('back')
  })

  it('turns nothing where the hand barely moved', () => {
    expect(swipeTurn(0)).toBeUndefined()
    expect(swipeTurn(SWIPE - 1)).toBeUndefined()
    expect(swipeTurn(1 - SWIPE)).toBeUndefined()
  })

  it('turns back at the near edge and on at the far one', () => {
    expect(pressTurn(0, WIDE)).toBe('back')
    expect(pressTurn(WIDE * EDGE, WIDE)).toBe('back')
    expect(pressTurn(WIDE, WIDE)).toBe('next')
    expect(pressTurn(WIDE * (1 - EDGE), WIDE)).toBe('next')
  })

  it('turns nothing on a press in the text itself', () => {
    expect(pressTurn(WIDE / 2, WIDE)).toBeUndefined()
    expect(pressTurn(10, 0)).toBeUndefined()
  })
})

describe('a hand put down and lifted', () => {
  it('turns the way it was drawn', () => {
    expect(handTurn(500, 500 - SWIPE, WIDE, false)).toBe('next')
    expect(handTurn(500, 500 + SWIPE, WIDE, false)).toBe('back')
  })

  it('turns on a press near either edge, having gone nowhere', () => {
    expect(handTurn(10, 10, WIDE, false)).toBe('back')
    expect(handTurn(WIDE - 10, WIDE - 10, WIDE, false)).toBe('next')
  })

  it('turns nothing where words were taken up', () => {
    // A run worth quoting is wider than a swipe, and taking one up ends over
    // the edge as often as not.
    expect(handTurn(500, 500 - SWIPE, WIDE, true)).toBeUndefined()
    expect(handTurn(WIDE - 10, WIDE - 10, WIDE, true)).toBeUndefined()
    expect(handTurn(10, 10, WIDE, true)).toBeUndefined()
  })
})

describe('the page a person is looking at', () => {
  const flow = { along: 10 * WIDE, width: WIDE, gap: GAP, columns: 2 }
  const document = { from: 2000, to: 3000 }
  const book = { from: 0, to: 10_000 }

  /** A run standing in each column the text was laid into. */
  const createMarks = (columns: number): Mark[] =>
    Array.from({ length: columns }, (_, column) => ({
      at: document.from + column,
      x: column * (columnWidth(flow) + GAP),
    }))

  const marks = createMarks(columnsInAll(flow))
  const here = marks.length
  const perColumn = (document.to - document.from) / here
  const before = Math.round((document.from - book.from) / perColumn)

  it('counts the columns on the screen, so a spread of two turns two pages', () => {
    const first = pagesOf(book, document, flow, 0, marks)
    const next = pagesOf(book, document, flow, 1, marks)
    expect(next.page - first.page).toBe(2)
  })

  it('counts the columns before this document at what a column of it holds', () => {
    expect(pagesOf(book, document, flow, 0, marks).page).toBe(before + 1)
    expect(pagesOf(book, document, flow, 0, marks).pages).toBe(
      Math.round((book.to - book.from) / perColumn),
    )
  })

  it('counts the columns the text fills and not the boxes drawn beside them', () => {
    // A document of one line stands in one column of a spread of two.
    const one = createMarks(1)
    expect(countFilledColumns(one, flow)).toBe(1)
    expect(pagesOf({ ...document }, document, flow, 0, one)).toEqual({ page: 1, pages: 1 })
  })

  it('says one page for a document nothing has been laid out for', () => {
    expect(pagesOf(book, document, flow, 0, [])).toEqual({ page: 1, pages: 1 })
  })
})

describe('what is left of the document on the page in front', () => {
  const flow: Flow = { along: 2700, width: 800, gap: GAP, columns: 2 }
  const marks: readonly Mark[] = [
    { at: 0, x: 10 },
    { at: 400, x: 1610 },
  ]

  it('is the columns of it standing past the spread in front', () => {
    expect(leftInDocument(flow, 0, marks)).toBe(3)
  })

  it('is nothing once the spread in front is the last that holds any', () => {
    // Five columns hold the text, and the third spread carries the last of
    // them: past it the document owes nobody anything.
    expect(leftInDocument(flow, 2, marks)).toBe(0)
  })
})

describe('the words the page count is said in', () => {
  it('put the page in front against all of them', () => {
    expect(BOOK_WORDS.of(3, 40)).toBe('3 of 40')
  })

  it('say how much of the chapter is still to come, one page or many', () => {
    expect(BOOK_WORDS.left(1)).toBe('1 page left in chapter')
    expect(BOOK_WORDS.left(5)).toBe('5 pages left in chapter')
  })
})

describe('where a turn lands', () => {
  const span = { from: 400, to: 900 }
  const book = { from: 0, to: 1400 }

  it('lands on the spread the turn asks for, inside the document', () => {
    expect(turnTo('next', 0, 3, span, book)).toStrictEqual({ spread: 1 })
    expect(turnTo('back', 2, 3, span, book)).toStrictEqual({ spread: 1 })
    expect(turnTo('first', 1, 3, span, book)).toStrictEqual({ spread: 0 })
    expect(turnTo('last', 1, 3, span, book)).toStrictEqual({ spread: 2 })
  })

  it('asks for the document beside this one, past either end of it', () => {
    expect(turnTo('next', 2, 3, span, book)).toStrictEqual({ offset: 900 })
    expect(turnTo('back', 0, 3, span, book)).toStrictEqual({ offset: 399 })
  })

  it('lands nowhere at either end of the book itself', () => {
    expect(turnTo('next', 2, 3, span, span)).toStrictEqual({})
    expect(turnTo('back', 0, 3, span, span)).toStrictEqual({})
  })
})
