/**
 * Where the countSpreads of a book stand when its text is set in columns.
 *
 * Apart from the component the way the reader's own arithmetic is: how many
 * countSpreads a laid-out document evaluates to, where each one begins, which one a run
 * of text falls in and how far into a run a byte offset reaches are arithmetic,
 * and a test asks them without a browser.
 */
import type { Span } from '@/shared/lib/text/span'

/** Where one run of the text stands, once the document is laid out. */
export interface Mark {
  /** The byte offset the run begins at, read off `data-offset`. */
  readonly at: number
  /** Position along the columns, in CSS pixels from the first column. */
  readonly x: number
}

/** A document set in columns, measured. */
export interface Flow {
  /** Total rendered columns width, in CSS pixels. */
  readonly contentWidth: number
  /** The reading area one spread fills, in CSS pixels. */
  readonly width: number
  /** Spacing between two columns, in CSS pixels. */
  readonly gap: number
  /** How many columns stand in one spread. */
  readonly columns: number
}

/**
 * Spacing between two columns, in CSS pixels. Half spacing is applied at outer
 * edges, so outer columns keep the same distance from container borders.
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
export function countColumns(width: number, size: number): number {
  return width >= 2 * (NARROWEST * size + GAP) ? 2 : 1
}

/**
 * Column height calculated to fit full lines.
 */
export function columnHeight(height: number, line: number): number {
  if (line <= 0 || height <= 0) return Math.max(height, 0)
  return Math.max(Math.floor(height / line), 1) * line
}

/**
 * Distance between consecutive columns.
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
 * How many columns the text evaluates to. The count is taken off the whole run and
 * rounded once: a trailing gap adds no column, and nothing drifts over a long
 * document.
 */
export function columnsInAll(flow: Flow): number {
  const pitch = columnPitch(flow)
  if (pitch <= 0 || flow.contentWidth <= 0) return 0
  return Math.max(1, Math.round(flow.contentWidth / pitch))
}

/**
 * Which column a place along them falls in, counted from the first. A column
 * begins half a gap into its share of the area, and the browser lays the
 * columns out in whole device pixels, so that half gap is also what a run
 * measured a fraction of a pixel short of its own column is read against.
 */
export function getColumn(flow: Flow, x: number): number {
  const pitch = columnPitch(flow)
  if (pitch <= 0) return 0
  return Math.max(Math.floor(x / pitch), 0)
}

/**
 * How many columns the text of this document actually fills. The layout renders
 * as many column boxes as a spread has, so a single-line document occupies one
 * column beside an empty one, and only the marks indicate which.
 */
export function countFilledColumns(marks: readonly Mark[], flow: Flow): number {
  if (columnWidth(flow) <= 0 || marks.length === 0) return 0
  let last = 0
  for (const mark of marks) last = Math.max(last, getColumn(flow, mark.x))
  return last + 1
}

/** Reading position: active column index and total column count. */
export interface Pages {
  readonly page: number
  readonly pages: number
}

/**
 * Number of columns of this document after the active spread.
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
 * What the rest of the book evaluates to is that document's own bytes to the column
 * carried over it — an estimate, and the only one that costs nothing. It moves
 * when the reading area does, which is what a page measured on the screen does.
 */
export interface PagesQuery {
  readonly book: Span
  readonly document: Span
  readonly flow: Flow
  readonly spread: number
  readonly marks: readonly Mark[]
}

export function createPages(query: PagesQuery): Pages {
  const { book, document, flow, spread, marks } = query
  const here = countFilledColumns(marks, flow)
  const first = spread * flow.columns + 1
  const bytes = document.to - document.from
  if (bytes <= 0) return { page: Math.max(first, 1), pages: Math.max(here, 1) }

  const perColumn = bytes / here
  const before = Math.round((document.from - book.from) / perColumn)
  const all = Math.round((book.to - book.from) / perColumn)
  return {
    page: Math.max(before + first, 1),
    pages: Math.max(all, before + here, 1),
  }
}

/** How many countSpreads the text evaluates to. */
export function countSpreads(flow: Flow): number {
  const all = columnsInAll(flow)
  return all === 0 ? 0 : Math.ceil(all / flow.columns)
}

/**
 * Computes spread starting horizontal offset in CSS pixels.
 */
export function getSpreadStart(flow: Flow, spread: number): number {
  return spread * flow.width
}

/** Which spread a place along the columns falls in. */
export function getSpread(flow: Flow, x: number): number {
  if (columnWidth(flow) <= 0) return 0
  const last = Math.max(countSpreads(flow) - 1, 0)
  return Math.min(Math.floor(getColumn(flow, x) / flow.columns), last)
}

/**
 * Returns the byte offset of the first mark in a spread, or undefined if empty.
 */
export function inFront(marks: readonly Mark[], flow: Flow, spread: number): number | undefined {
  for (const mark of marks) {
    if (getSpread(flow, mark.x) === spread) return mark.at
  }
  return undefined
}

/** Finds the spread index containing or preceding the given byte offset. */
export function findSpreadAt(marks: readonly Mark[], flow: Flow, offset: number): number {
  let found: Mark | undefined
  for (const mark of marks) {
    if (mark.at > offset) break
    found = mark
  }
  return found ? getSpread(flow, found.x) : 0
}

/** How large the text may be set. */
export const SMALLEST = 0.8
export const LARGEST = 2
