/**
 * Where the countSpreads of a book stand when its text is set in columns.
 *
 * Apart from the component the way the reader's own arithmetic is: how many
 * countSpreads a laid-out document comes to, where each one begins, which one a run
 * of text falls in and how far into a run a byte offset reaches are arithmetic,
 * and a test asks them without a browser.
 */
import type { Span } from '@/shared/lib/span'

/** Where one run of the text stands, once the document is laid out. */
export interface Mark {
  /** The byte offset the run begins at, read off `data-offset`. */
  readonly at: number
  /** Where it stands along the columns, in CSS pixels from the first of them. */
  readonly x: number
}

/** A document set in columns, measured. */
export interface Flow {
  /** How far the columns run, in CSS pixels. */
  readonly along: number
  /** The reading area one spread fills, in CSS pixels. */
  readonly width: number
  /** What stands between two columns, in CSS pixels. */
  readonly gap: number
  /** How many columns stand in one spread. */
  readonly columns: number
}

/**
 * What stands between two columns, in CSS pixels. Half of it stands beside the
 * outermost column at either edge, so a column keeps the same distance from the
 * one beside it as two columns of a spread keep from each other, and a spread
 * begins every reading area's width along.
 */
export const GAP = 72

/**
 * The narrowest a column is set, in CSS pixels at the size the text is read at.
 * A line of about fifty-six characters, which is within the measure prose is
 * set at;
 * a second column is opened only where both can be read at it.
 */
export const NARROWEST = 448

/**
 * How many columns a reading area holds. Two once each of them can be set at
 * the narrowest a column is read at, and one until then.
 */
export function columnsIn(width: number, size: number): number {
  return width >= 2 * (NARROWEST * size + GAP) ? 2 : 1
}

/**
 * How tall a column is set: a whole number of lines, and never fewer than one.
 * A column set to the height of the area it stands in ends part of the way
 * through a line, and the half of it below the edge is cut off.
 */
export function columnHeight(height: number, line: number): number {
  if (line <= 0 || height <= 0) return Math.max(height, 0)
  return Math.max(Math.floor(height / line), 1) * line
}

/**
 * How far one column stands from the next, which is the reading area shared out
 * between the columns of a spread.
 */
export function columnPitch(flow: Flow): number {
  if (flow.columns <= 0) return 0
  return flow.width / flow.columns
}

/** How wide one column is set: its share of the area, less the gap it keeps. */
export function columnWidth(flow: Flow): number {
  return Math.max(columnPitch(flow) - flow.gap, 0)
}

/**
 * How many columns the text comes to. The count is taken off the whole run and
 * rounded once: a trailing gap adds no column, and nothing drifts over a long
 * document.
 */
export function columnsInAll(flow: Flow): number {
  const pitch = columnPitch(flow)
  if (pitch <= 0 || flow.along <= 0) return 0
  return Math.max(1, Math.round(flow.along / pitch))
}

/**
 * Which column a place along them falls in, counted from the first. A column
 * begins half a gap into its share of the area, and the browser lays the
 * columns out in whole device pixels, so that half gap is also what a run
 * measured a fraction of a pixel short of its own column is read against.
 */
export function columnAt(flow: Flow, x: number): number {
  const pitch = columnPitch(flow)
  if (pitch <= 0) return 0
  return Math.max(Math.floor(x / pitch), 0)
}

/**
 * How many columns the text of this document actually fills. The layout draws
 * as many column boxes as a spread has, so a document of one line stands in one
 * column and beside an empty one, and only the runs say which.
 */
export function countFilledColumns(marks: readonly Mark[], flow: Flow): number {
  if (columnWidth(flow) <= 0 || marks.length === 0) return 0
  let last = 0
  for (const mark of marks) last = Math.max(last, columnAt(flow, mark.x))
  return last + 1
}

/** Where a person stands in a book: the column in front, and how many there are. */
export interface Pages {
  readonly page: number
  readonly pages: number
}

/**
 * How many columns of this document stand after the spread in front. The
 * document is laid out, so what is left of it is known exactly.
 */
export function leftInDocument(flow: Flow, spread: number, marks: readonly Mark[]): number {
  const here = countFilledColumns(marks, flow)
  return Math.max(here - (spread + 1) * flow.columns, 0)
}

/**
 * The page in front and how many the book is read in, counted in the columns a
 * person is looking at: a spread of two turns two pages.
 *
 * Only the document being read is laid out, so only its columns are counted.
 * What the rest of the book comes to is that document's own bytes to the column
 * carried over it — an estimate, and the only one that costs nothing. It moves
 * when the reading area does, which is what a page measured on the screen does.
 */
export function pagesOf(
  book: Span,
  document: Span,
  flow: Flow,
  spread: number,
  marks: readonly Mark[],
): Pages {
  const here = countFilledColumns(marks, flow)
  const first = spread * flow.columns + 1
  const bytes = document.to - document.from
  if (here <= 0 || bytes <= 0) return { page: Math.max(first, 1), pages: Math.max(here, 1) }

  const perColumn = bytes / here
  const before = Math.round((document.from - book.from) / perColumn)
  const all = Math.round((book.to - book.from) / perColumn)
  return {
    page: Math.max(before + first, 1),
    pages: Math.max(all, before + here, 1),
  }
}

/** How many countSpreads the text comes to. */
export function countSpreads(flow: Flow): number {
  const all = columnsInAll(flow)
  return all === 0 ? 0 : Math.ceil(all / flow.columns)
}

/**
 * Where a spread begins along the columns, in CSS pixels. The gap a column
 * keeps stands inside the area at either edge, so one spread begins the whole
 * of a reading area along from the one before it.
 */
export function getSpreadStart(flow: Flow, spread: number): number {
  return spread * flow.width
}

/** Which spread a place along the columns falls in. */
export function spreadAt(flow: Flow, x: number): number {
  if (columnWidth(flow) <= 0) return 0
  const last = Math.max(countSpreads(flow) - 1, 0)
  return Math.min(Math.floor(columnAt(flow, x) / flow.columns), last)
}

/**
 * The offset of the first run standing in a spread, and nothing where no run
 * stands there. The marks are in the order the text is.
 */
export function inFront(marks: readonly Mark[], flow: Flow, spread: number): number | undefined {
  for (const mark of marks) {
    if (spreadAt(flow, mark.x) === spread) return mark.at
  }
  return undefined
}

/** The spread an offset stands in: that of the last run beginning at or before it. */
export function findSpreadAt(marks: readonly Mark[], flow: Flow, at: number): number {
  let found: Mark | undefined
  for (const mark of marks) {
    if (mark.at > at) break
    found = mark
  }
  return found ? spreadAt(flow, found.x) : 0
}

/** How large the text may be set. */
export const SMALLEST = 0.8
export const LARGEST = 2

/** A number held inside the bounds it is read between. */
export const clamp = (value: number, least: number, most: number): number =>
  Math.min(Math.max(value, least), most)
