/**
 * Where the pages of a document stand when they are laid in a row.
 *
 * Apart from the component the way `plex/arrange` is: where a page begins, which
 * pages are in the viewport and which one is in front are arithmetic, and a test
 * asks them without a browser.
 */
import type { Size } from '@/shared/lib/geometry'

/** Where something sits on a page, in fractions of it. */
export interface Rect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** The words a document is read with, declared once. */
export interface ReaderWords {
  /** What turning back a page is called, and turning on. */
  readonly back: string
  readonly next: string
  /** What the field the page is typed in is called. */
  readonly page: string
  /** What the row of pages is called, which the keyboard scrolls. */
  readonly pages: string
  /** What drawing the page larger is called, and smaller. */
  readonly closer: string
  readonly further: string
}

export const READER_WORDS: ReaderWords = {
  back: 'Previous page',
  next: 'Next page',
  page: 'Page',
  pages: 'Pages',
  closer: 'Closer',
  further: 'Further',
}

/** One page's size, in the page's own units. */
export interface Page {
  readonly width: number
  readonly height: number
}

/** What stands between two pages, and around the row, in CSS pixels. */
export const GAP = 16

/**
 * The shape of a page nothing said the size of. A document answers with its
 * pages' sizes, and one that has not answered yet still has to be laid out.
 */
export const UPRIGHT: Page = { width: 612, height: 792 }

/**
 * How much beyond the edge of the viewport is drawn, as a share of it. A page
 * turned to is drawn before it is reached, and a page just left is kept in case
 * the hand comes back.
 */
const BEYOND = 1.5

/**
 * A row of pages: how tall they are drawn, where each one begins, and what the
 * whole row comes to.
 *
 * Every page is drawn to one height, so the row stands on one line however the
 * pages are shaped, and a flick sideways is a page.
 */
export interface Row {
  /** How tall every page is drawn, in CSS pixels. */
  readonly height: number
  /** Where each page begins along the row, in CSS pixels. */
  readonly starts: readonly number[]
  /** How wide each page is drawn, in CSS pixels. */
  readonly widths: readonly number[]
  /** What the whole row comes to, in CSS pixels. */
  readonly length: number
}

/**
 * The row a document makes in a viewport, drawn `zoom` times the size at which
 * a whole page stands in that viewport.
 *
 * A page whose size is not known takes the first page's, and a document that
 * has said nothing takes an upright page. Laying the row out on nothing would
 * put every page at the same place, and the row would jump as the sizes came.
 *
 * A viewport with no height makes no row. Until something has been measured
 * there is no width to draw a page at, and a page drawn at a made-up one is a
 * page drawn and thrown away.
 */
export function row(pages: readonly Page[], viewport: Size, zoom: number): Row {
  const height = Math.round((viewport.height - 2 * GAP) * zoom)
  if (height <= 0) return { height: 0, starts: [], widths: [], length: 0 }
  const starts: number[] = []
  const widths: number[] = []
  let along = GAP
  for (let page = 0; page < pages.length; page++) {
    const size = sizeOf(pages, page)
    const width = Math.max(Math.round((height * size.width) / size.height), 1)
    starts.push(along)
    widths.push(width)
    along += width + GAP
  }
  return { height, starts, widths, length: along }
}

/** The size of one page, and the nearest thing to it that is known. */
function sizeOf(pages: readonly Page[], page: number): Page {
  const said = pages[page]
  if (said && said.width > 0 && said.height > 0) return said
  const first = pages[0]
  return first && first.width > 0 && first.height > 0 ? first : UPRIGHT
}

/**
 * The pages to draw: those in the viewport, and a little either side of it.
 *
 * A book is five hundred pages and a page is half a megabyte. A row that drew
 * all of them would ask for a book's worth of pixels to show one page.
 */
export function getPagesWithin(row: Row, viewport: Size, along: number): number[] {
  const from = along - viewport.width * BEYOND
  const to = along + viewport.width * (1 + BEYOND)
  const out: number[] = []
  for (let page = 0; page < row.starts.length; page++) {
    const begins = row.starts[page]!
    if (begins > to) break
    if (begins + row.widths[page]! >= from) out.push(page)
  }
  return out
}

/**
 * The page in front: the one under the middle of the viewport. A page scrolled
 * halfway off is not the page a person is reading.
 */
export function inFront(row: Row, viewport: Size, along: number): number {
  const at = along + viewport.width / 2
  let page = 0
  for (let i = 0; i < row.starts.length; i++) {
    if (row.starts[i]! > at) break
    page = i
  }
  return page
}

/** How far the row is scrolled to put one page against the left edge. */
export function standAt(row: Row, page: number): number | undefined {
  const begins = row.starts[page]
  return begins === undefined ? undefined : begins - GAP
}

/**
 * How close a page may be drawn. One is a whole page in the viewport it is read
 * in, which is the furthest there is: a page smaller than the viewport it
 * stands in is room going to waste.
 */
export const FURTHEST = 1
export const CLOSEST = 6

/** How much closer one press draws the page. */
export const NEARER = 1.25

/**
 * The widths a page is asked for, in device pixels. Dragging the edge of a pane
 * crosses a few of them, and a page drawn wider than its box is drawn down into
 * it.
 */
export const STAGE = 128

/**
 * How long a width has to have stood still before a page is asked for at it, in
 * milliseconds. A pane edge dragged across a screen crosses a dozen widths, and
 * each one is a page drawn and thrown away.
 */
export const SETTLED = 150

/** How close a page is drawn, never past either end. */
export function clampZoom(zoom: number): number {
  return Math.min(Math.max(zoom, FURTHEST), CLOSEST)
}
