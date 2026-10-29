/**
 * Page positions when the pages of a document are laid out in a row.
 *
 * Separate from the component, like `plex/arrange`: where a page begins, which
 * pages are in the viewport and which one is in front are arithmetic, and a
 * test evaluates them without a browser.
 */
import type { Size } from '@numen/ui'
import { getPageNavigationAction } from '@numen/ui'

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
  readonly zoomIn: string
  readonly zoomOut: string
}

export const READER_WORDS: ReaderWords = {
  back: 'Previous page',
  next: 'Next page',
  page: 'Page',
  pages: 'Pages',
  zoomIn: 'Closer',
  zoomOut: 'Further',
}

/** One page's size, in the page's own units. */
export interface Page {
  readonly width: number
  readonly height: number
}

export const GAP = 16

/**
 * The default page size, used for pages whose size is unknown. A document
 * reports its page sizes, and a document that has not reported yet is still laid out.
 */
export const UPRIGHT: Page = { width: 612, height: 792 }

/**
 * How far beyond the viewport edge pages are drawn, as a fraction of the
 * viewport. A target page is drawn before it is reached, and a page just left
 * stays drawn in case the scroll reverses.
 */
const BEYOND = 1.5

/**
 * A row of pages: how tall they are drawn, where each one begins, and what the
 * whole row evaluates to.
 *
 * Every page is drawn to one height, so the row relies on one line however the
 * pages are shaped, and a flick sideways is a page.
 */
export interface Row {
  /** How tall every page is drawn, in CSS pixels. */
  readonly height: number
  /** Where each page begins along the row, in CSS pixels. */
  readonly starts: readonly number[]
  /** How wide each page is drawn, in CSS pixels. */
  readonly widths: readonly number[]
  /** What the whole row evaluates to, in CSS pixels. */
  readonly length: number
}

/**
 * The row a document forms in a viewport, drawn at `zoom` times the base size.
 *
 * A page of unknown size takes the first page's size, and a document with no
 * reported sizes uses the default page size. Laying out the row with no sizes
 * would place every page at the same position, and the row would shift as the
 * sizes arrive.
 *
 * A viewport with no height forms no row. Until a size is measured there is no
 * width to draw a page at, and a page drawn at an arbitrary width is discarded.
 */
export function row(pages: readonly Page[], viewport: Size, zoom: number): Row {
  const height = Math.round((viewport.height - 2 * GAP) * zoom)
  if (height <= 0) return { height: 0, starts: [], widths: [], length: 0 }
  const starts: number[] = []
  const widths: number[] = []
  let axisOffset = GAP
  for (let page = 0; page < pages.length; page++) {
    const size = getSize(pages, page)
    const width = Math.max(Math.round((height * size.width) / size.height), 1)
    starts.push(axisOffset)
    widths.push(width)
    axisOffset += width + GAP
  }
  return { height, starts, widths, length: axisOffset }
}

/** The size of one page, and the nearest thing to it that is known. */
function getSize(pages: readonly Page[], page: number): Page {
  const targetPage = pages[page]
  if (targetPage && targetPage.width > 0 && targetPage.height > 0) return targetPage
  const first = pages[0]
  return first && first.width > 0 && first.height > 0 ? first : UPRIGHT
}

/**
 * The pages to draw: those in the viewport and a margin on either side.
 *
 * A book has five hundred pages at half a megabyte each. Drawing all of them
 * would allocate a book's worth of pixels to show one page.
 */
export function getPagesWithin(row: Row, viewport: Size, offset: number): number[] {
  const from = offset - viewport.width * BEYOND
  const to = offset + viewport.width * (1 + BEYOND)
  const out: number[] = []
  for (const [page, begins] of row.starts.entries()) {
    if (begins > to) break
    if (begins + row.widths[page]! >= from) out.push(page)
  }
  return out
}

/**
 * The page in front: the one at the middle of the viewport. A page scrolled
 * halfway out of view is not the page being read.
 */
export function inFront(row: Row, viewport: Size, offset: number): number {
  const center = offset + viewport.width / 2
  let page = 0
  for (let i = 0; i < row.starts.length; i++) {
    if (row.starts[i]! > center) break
    page = i
  }
  return page
}

/** How far the row is scrolled to put one page against the left edge. */
export function findPageScroll(row: Row, page: number): number | undefined {
  const begins = row.starts[page]
  return begins === undefined ? undefined : begins - GAP
}

/**
 * How far the row is scrolled to bring a page against the left edge, held
 * inside the row: the last page cannot be brought further left than the end of
 * it. Nothing for a page the row does not hold.
 */
export function getScrollTarget(row: Row, viewport: Size, page: number): number | undefined {
  const begins = findPageScroll(row, page)
  if (begins === undefined) return undefined
  const getFurthest = Math.max(row.length - viewport.width, 0)
  return Math.min(Math.max(begins, 0), getFurthest)
}

/**
 * The page a key turns to, counted from the first. Nothing for a key that turns
 * no page, a document with no pages, or a turn that leaves the document.
 */
export function getTurnTarget(key: string, pos: number, pageCount: number): number | undefined {
  const way = getPageNavigationAction(key)
  if (!way || pageCount === 0) return undefined

  const last = pageCount - 1
  let to: number
  if (way === 'first') to = 0
  else if (way === 'last') to = last
  else to = pos + (way === 'next' ? 1 : -1)

  return to < 0 || to > last ? undefined : to
}

/**
 * How close a page may be drawn. One is a whole page in the viewport it is read
 * in, which is the getFurthest there is: a page smaller than the viewport it
 */
export const FURTHEST = 1
export const CLOSEST = 6

/** How much closer one press draws the page. */
export const NEARER = 1.25

/**
 * The quantization step for requested page widths, in device pixels. Dragging a
 * pane edge changes the width continuously, and quantizing limits the number of
 * distinct widths requested.
 */
export const WIDTH_STEP = 128

/**
 * How long a width must stay unchanged before a page is requested at it, in
 * milliseconds. Dragging a pane edge across the screen crosses a dozen widths,
 * and each would produce a page that is drawn and discarded.
 */
export const WIDTH_DEBOUNCE = 150

/** How close a page is drawn, never past either end. */
export function clampZoom(zoom: number): number {
  return Math.min(Math.max(zoom, FURTHEST), CLOSEST)
}
