/**
 * Where the spreads of a book stand when its text is set in columns.
 *
 * Apart from the component the way `reader/strip` is: how many spreads a
 * laid-out document comes to, where each one begins, which one a run of text
 * falls in and how far into a run a byte offset reaches are arithmetic, and a
 * test asks them without a browser.
 */
import { READER_WORDS, type ReaderWords } from '@/reader/strip'

/** A stretch of the book's text, in bytes of it. */
export interface Span {
  readonly begins: number
  readonly ends: number
}

/** Where one run of the text stands, once the document is laid out. */
export interface Mark {
  /** The byte offset the run begins at, read off `data-at`. */
  readonly at: number
  /** Where it stands along the columns, in CSS pixels from the first of them. */
  readonly x: number
}

/** A document set in columns, measured. */
export interface Flow {
  /** How far the columns run, in CSS pixels. */
  readonly along: number
  /** The reading area one spread fills, in CSS pixels. */
  readonly wide: number
  /** What stands between two columns, in CSS pixels. */
  readonly gap: number
  /** How many columns stand in one spread. */
  readonly columns: number
}

/** What stands between two columns, in CSS pixels. */
export const GAP = 32

/**
 * The narrowest a column is set, in CSS pixels at the size the text is read at.
 * Around fifty characters, which is the short end of a line a person can read
 * without losing their place in it.
 */
export const NARROWEST = 336

/**
 * How many columns a reading area holds. Two once each of them can be set at
 * the narrowest a column is read at, and one until then.
 */
export function columnsIn(wide: number, size: number): number {
  return wide >= 2 * NARROWEST * size + GAP ? 2 : 1
}

/** How wide one column is set. */
export function columnWide(flow: Flow): number {
  if (flow.columns <= 0) return 0
  return (flow.wide - (flow.columns - 1) * flow.gap) / flow.columns
}

/**
 * How many columns the text comes to. The count is taken off the whole run and
 * rounded once: a trailing gap adds no column, and nothing drifts over a long
 * document.
 */
export function columnsInAll(flow: Flow): number {
  const one = columnWide(flow)
  if (one <= 0 || flow.along <= 0) return 0
  return Math.max(1, Math.round((flow.along + flow.gap) / (one + flow.gap)))
}

/** How many spreads the text comes to. */
export function spreads(flow: Flow): number {
  const all = columnsInAll(flow)
  return all === 0 ? 0 : Math.ceil(all / flow.columns)
}

/**
 * Where a spread begins along the columns, in CSS pixels. Each one is a whole
 * number of spreads out from the first, and one gap stands between two spreads
 * as it stands between two columns.
 */
export function beginsAt(flow: Flow, spread: number): number {
  return spread * (flow.wide + flow.gap)
}

/** Which spread a place along the columns falls in. */
export function spreadAt(flow: Flow, x: number): number {
  const one = columnWide(flow)
  if (one <= 0) return 0
  const column = Math.floor((x + flow.gap / 2) / (one + flow.gap))
  const last = Math.max(spreads(flow) - 1, 0)
  return Math.min(Math.max(Math.floor(column / flow.columns), 0), last)
}

/**
 * The offset of the first run standing in a spread, and nothing where no run
 * stands there. The marks are in the order the text is.
 */
export function inFront(
  marks: readonly Mark[],
  flow: Flow,
  spread: number,
): number | undefined {
  for (const mark of marks) {
    if (spreadAt(flow, mark.x) === spread) return mark.at
  }
  return undefined
}

/** The spread an offset stands in: that of the last run beginning at or before it. */
export function holding(marks: readonly Mark[], flow: Flow, at: number): number {
  let found: Mark | undefined
  for (const mark of marks) {
    if (mark.at > at) break
    found = mark
  }
  return found ? spreadAt(flow, found.x) : 0
}

const encoder = new TextEncoder()

/** How many bytes a text comes to. */
export function bytesIn(text: string): number {
  return encoder.encode(text).length
}

/**
 * How many UTF-16 units of a text its first so many bytes cover.
 *
 * The offsets a book carries are bytes and a JavaScript string is units. A
 * character of Devanagari is three bytes and one of Cyrillic is two, so the two
 * numbers part company on the first word of the corpus this reads.
 */
export function unitsIn(text: string, bytes: number): number {
  if (bytes <= 0) return 0
  let counted = 0
  let units = 0
  for (const character of text) {
    const size = encoder.encode(character).length
    if (counted + size > bytes) break
    counted += size
    units += character.length
  }
  return units
}

/** How large the text may be set, and how much larger one press sets it. */
export const SMALLEST = 0.8
export const LARGEST = 2
export const LARGER = 1.125

/** The words a book is read with. */
export const BOOK_WORDS: ReaderWords = {
  ...READER_WORDS,
  closer: 'Larger',
  further: 'Smaller',
}

/** A turn of the page, and the two ends of the document. */
export type PageTurn = 'back' | 'next' | 'first' | 'last'

/** Which way a key turns the page, and nothing for a key that turns none. */
export function keyTurn(key: string): PageTurn | undefined {
  switch (key) {
    case 'ArrowRight':
    case 'ArrowDown':
    case 'PageDown':
    case ' ':
      return 'next'
    case 'ArrowLeft':
    case 'ArrowUp':
    case 'PageUp':
      return 'back'
    case 'Home':
      return 'first'
    case 'End':
      return 'last'
    default:
      return undefined
  }
}

/** How far a hand travels sideways before it is a swipe, in CSS pixels. */
export const SWIPE = 40

/** Which way a swipe turns the page: the page follows the hand. */
export function swipeTurn(by: number): PageTurn | undefined {
  if (by <= -SWIPE) return 'next'
  if (by >= SWIPE) return 'back'
  return undefined
}

/** How much of either edge of the reading area is a press that turns, as a share of it. */
export const EDGE = 0.15

/** Which way a press across the reading area turns the page. */
export function pressTurn(x: number, wide: number): PageTurn | undefined {
  if (wide <= 0) return undefined
  if (x <= wide * EDGE) return 'back'
  if (x >= wide * (1 - EDGE)) return 'next'
  return undefined
}
