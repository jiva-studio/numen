import { describe, expect, it } from 'vitest'
import {
  EDGE,
  GAP,
  NARROWEST,
  SWIPE,
  beginsAt,
  bytesIn,
  columnWide,
  columnsIn,
  columnsInAll,
  holding,
  inFront,
  keyTurn,
  handTurn,
  pressTurn,
  spreadAt,
  spreads,
  swipeTurn,
  unitsIn,
  type Flow,
  type Mark,
} from './spread'

/** A wide reading area, which takes two columns, and a narrow one, which takes one. */
const WIDE = 800
const NARROW = 420

/**
 * The columns a document of so many of them comes to, in a given reading area.
 * A column stands one gap after the one before it, which is how the browser
 * lays them out and how far a spread has to move.
 */
const laid = (wide: number, columns: number, all: number): Flow => {
  const one = (wide - (columns - 1) * GAP) / columns
  return { along: all * one + (all - 1) * GAP, wide, gap: GAP, columns }
}

describe('how many columns a reading area takes', () => {
  it('takes two once each of them can be set at the narrowest a column is read at', () => {
    expect(columnsIn(2 * NARROWEST + GAP, 1)).toBe(2)
    expect(columnsIn(2 * NARROWEST + GAP - 1, 1)).toBe(1)
  })

  it('takes one in an area nothing has been measured in', () => {
    expect(columnsIn(0, 1)).toBe(1)
  })

  it('takes the size the text is set at into account', () => {
    // A column is measured in characters, so text set half again as large needs
    // half again as much room before a second column will hold a line.
    const wide = 2 * NARROWEST + GAP
    expect(columnsIn(wide, 1)).toBe(2)
    expect(columnsIn(wide, 1.5)).toBe(1)
  })

  it('sets each column to its share of what is left after the gap', () => {
    expect(columnWide({ along: 0, wide: WIDE, gap: GAP, columns: 2 })).toBe((WIDE - GAP) / 2)
    expect(columnWide({ along: 0, wide: NARROW, gap: GAP, columns: 1 })).toBe(NARROW)
  })
})

describe('how many spreads a document comes to', () => {
  it('counts the columns off the whole run', () => {
    expect(columnsInAll(laid(WIDE, 2, 10))).toBe(10)
    expect(columnsInAll(laid(NARROW, 1, 7))).toBe(7)
  })

  it('turns them into spreads, two columns to a spread in a wide area', () => {
    expect(spreads(laid(WIDE, 2, 10))).toBe(5)
    expect(spreads(laid(WIDE, 2, 9))).toBe(5)
    expect(spreads(laid(NARROW, 1, 7))).toBe(7)
  })

  it('adds no spread for a gap left standing after the last column', () => {
    // A run measured one gap longer than the columns in it is the same
    // document, and a spread of nothing at the end of a book is a page a person
    // turns to and finds empty.
    const flow = laid(WIDE, 2, 10)
    expect(spreads({ ...flow, along: flow.along + GAP })).toBe(5)
  })

  it('adds no spread for the pixel a browser rounds a column to', () => {
    const flow = laid(WIDE, 2, 10)
    expect(spreads({ ...flow, along: flow.along + 0.5 })).toBe(5)
    expect(spreads({ ...flow, along: flow.along - 0.5 })).toBe(5)
  })

  it('comes to nothing for a document with no text at all', () => {
    expect(spreads({ along: 0, wide: WIDE, gap: GAP, columns: 2 })).toBe(0)
    expect(columnsInAll({ along: 0, wide: WIDE, gap: GAP, columns: 2 })).toBe(0)
  })

  it('comes to one spread for a document of one line', () => {
    expect(spreads(laid(WIDE, 2, 1))).toBe(1)
    expect(spreads(laid(NARROW, 1, 1))).toBe(1)
  })

  it('comes to nothing in an area nothing has been measured in', () => {
    expect(spreads({ along: 1000, wide: 0, gap: GAP, columns: 1 })).toBe(0)
  })
})

describe('where a spread begins', () => {
  it('stands a whole number of spreads out from the first', () => {
    const flow = laid(WIDE, 2, 200)

    // A hundred spreads out, the place is still exactly the place the hundredth
    // column pair begins at: nothing is added up along the way.
    const one = columnWide(flow)
    expect(beginsAt(flow, 100)).toBe(200 * (one + GAP))
  })

  it('leaves one gap between two spreads, as between two columns', () => {
    const flow = laid(WIDE, 2, 10)
    expect(beginsAt(flow, 1) - beginsAt(flow, 0)).toBe(WIDE + GAP)
  })
})

describe('which spread a place falls in', () => {
  it('is the spread the column at that place belongs to', () => {
    const flow = laid(WIDE, 2, 10)
    const step = columnWide(flow) + GAP

    expect(spreadAt(flow, 0)).toBe(0)
    expect(spreadAt(flow, step)).toBe(0)
    expect(spreadAt(flow, 2 * step)).toBe(1)
    expect(spreadAt(flow, 3 * step)).toBe(1)
    expect(spreadAt(flow, 8 * step)).toBe(4)
  })

  it('goes no further than the last spread there is', () => {
    const flow = laid(WIDE, 2, 10)
    expect(spreadAt(flow, 100_000)).toBe(4)
    expect(spreadAt(flow, -100)).toBe(0)
  })
})

describe('which run of the text is in front', () => {
  /** Runs an offset apart, each one column further along. */
  const runs = (flow: Flow, count: number): Mark[] => {
    const step = columnWide(flow) + GAP
    return Array.from({ length: count }, (_, index) => ({ at: index * 100, x: index * step }))
  }

  it('is the first run standing in the spread', () => {
    const flow = laid(WIDE, 2, 10)
    const marks = runs(flow, 10)

    expect(inFront(marks, flow, 0)).toBe(0)
    expect(inFront(marks, flow, 1)).toBe(200)
    expect(inFront(marks, flow, 4)).toBe(800)
  })

  it('is nothing where no run stands in the spread', () => {
    const flow = laid(WIDE, 2, 10)
    // A picture filling the third spread on its own, with no run of text in it.
    const marks: Mark[] = [{ at: 0, x: 0 }, { at: 500, x: beginsAt(flow, 3) }]

    expect(inFront(marks, flow, 2)).toBeUndefined()
  })

  it('is nothing at all where the document has no runs', () => {
    expect(inFront([], laid(WIDE, 2, 1), 0)).toBeUndefined()
  })
})

describe('which spread an offset stands in', () => {
  const flow = laid(WIDE, 2, 10)
  const step = columnWide(flow) + GAP
  const marks: Mark[] = [
    { at: 0, x: 0 },
    { at: 100, x: 3 * step },
    { at: 240, x: 7 * step },
  ]

  it('is the spread of the last run beginning at or before it', () => {
    expect(holding(marks, flow, 0)).toBe(0)
    expect(holding(marks, flow, 99)).toBe(0)
    expect(holding(marks, flow, 100)).toBe(1)
    expect(holding(marks, flow, 180)).toBe(1)
    expect(holding(marks, flow, 240)).toBe(3)
  })

  it('is the first spread for an offset before anything the document carries', () => {
    expect(holding(marks, flow, -5)).toBe(0)
    expect(holding([], flow, 900)).toBe(0)
  })

  it('is the spread of the last run for an offset past everything', () => {
    expect(holding(marks, flow, 10_000)).toBe(3)
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
  it('turns on and back on the keys a book is read with', () => {
    expect(keyTurn('ArrowRight')).toBe('next')
    expect(keyTurn(' ')).toBe('next')
    expect(keyTurn('PageDown')).toBe('next')
    expect(keyTurn('ArrowLeft')).toBe('back')
    expect(keyTurn('PageUp')).toBe('back')
    expect(keyTurn('Home')).toBe('first')
    expect(keyTurn('End')).toBe('last')
  })

  it('turns nothing on a key that is not one of them', () => {
    expect(keyTurn('a')).toBeUndefined()
    expect(keyTurn('Enter')).toBeUndefined()
    expect(keyTurn('Tab')).toBeUndefined()
  })

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
