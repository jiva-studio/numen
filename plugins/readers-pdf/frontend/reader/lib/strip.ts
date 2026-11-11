/**
 * Page positions when the pages of a document are laid out in a row.
 *
 * Separate from the component, like `plex/arrange`: where a page begins, which
 * pages are in the viewport and which one is in front are arithmetic, and a
 * test evaluates them without a browser.
 */
import type { Size } from '@numen/ui'
import type { Fit } from '../../types'
import { getPageNavigationAction } from '@numen/ui'

export interface Rect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** The words a document is read with, declared once. */
export interface ReaderWords {
  readonly back: string
  readonly next: string
  readonly page: string
  readonly pages: string
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
const UPRIGHT: Page = { width: 612, height: 792 }

/**
 * How far beyond the viewport edge pages are drawn, as a fraction of the
 * viewport. A target page is drawn before the scroll arrives at it, and a page
 * scrolled out of view stays drawn in case the scroll reverses.
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
  readonly height: number
  readonly starts: readonly number[]
  readonly widths: readonly number[]
  readonly length: number
}

/**
 * The row a document forms in a viewport, drawn at `zoom` times the base size.
 * A page of unknown size takes the first page's size, and a document with no reported sizes uses the default page size.
 * Laying out the row with no sizes would place every page at one position and shift when the sizes arrive.
 * A viewport with no height forms no row: with no measured size there is no width to draw a page at.
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
export function getPagesWithin(
  row: Row,
  viewport: Size,
  offset: number,
  beyond = BEYOND,
): number[] {
  const from = offset - viewport.width * beyond
  const to = offset + viewport.width * (1 + beyond)
  const out: number[] = []
  for (const [page, begins] of row.starts.entries()) {
    if (begins > to) break
    if (begins + row.widths[page]! >= from) out.push(page)
  }
  return out
}

/**
 * The current page: the last page that begins at or before the vertical center
 * line of the viewport. A center line in the gap after a page belongs to that
 * page.
 */
export function getCurrentPage(row: Row, viewport: Size, offset: number): number {
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
 * How far the row is scrolled to show a page, held inside the row. Nothing for a page the row does not hold.
 * The page goes against the left edge when the viewport's center line then falls on it, and is centered otherwise.
 * `getCurrentPage` at the returned offset is the page, unless the offset was clamped at an end of the row.
 */
export function getScrollTarget(row: Row, viewport: Size, page: number): number | undefined {
  const leftEdge = findPageScroll(row, page)
  const begins = row.starts[page]
  const width = row.widths[page]
  if (leftEdge === undefined || begins === undefined || width === undefined) return undefined
  const isCenterOnPage = leftEdge + viewport.width / 2 < begins + width
  const target = isCenterOnPage ? leftEdge : Math.round(begins + width / 2 - viewport.width / 2)
  const furthest = Math.max(row.length - viewport.width, 0)
  return Math.min(Math.max(target, 0), furthest)
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

/** CSS pixels per point of a page: 96 CSS pixels and 72 points to the inch. */
const PIXELS_PER_POINT = 96 / 72

/**
 * The zoom at which the first page is drawn to the fit in the viewport, never past CLOSEST. A
 * viewport narrower than the page draws it narrower than a whole page by height, so the zoom
 * can be below FURTHEST; only a zoom chosen with the toolbar is limited to that.
 */
export function getFitZoom(fit: Fit, pages: readonly Page[], viewport: Size): number {
  const fullHeight = viewport.height - 2 * GAP
  if (fit === 'page' || fullHeight <= 0 || viewport.width <= 2 * GAP) return FURTHEST
  const size = getSize(pages, 0)
  const drawnHeight =
    fit === 'width'
      ? ((viewport.width - 2 * GAP) * size.height) / size.width
      : size.height * PIXELS_PER_POINT
  return Math.min(drawnHeight / fullHeight, CLOSEST)
}

/**
 * How far beyond the viewport edge pages are requested at the width the row
 * settled on, as a fraction of the viewport. A page past it keeps what it shows,
 * so a zoom re-renders what can be seen and what is about to be, not the strip.
 */
export const SHARP = 0.5

/**
 * A point in the row: a page and how far along it, as a fraction of its width.
 * The point survives a zoom, which changes every page width and every start.
 */
export interface Anchor {
  readonly page: number
  readonly fraction: number
}

/** The point of the row under the center line of the viewport. */
export function getAnchor(row: Row, viewport: Size, offset: number): Anchor | undefined {
  if (row.starts.length === 0) return undefined
  const page = getCurrentPage(row, viewport, offset)
  const width = row.widths[page]!
  return { page, fraction: (offset + viewport.width / 2 - row.starts[page]!) / width }
}

/** How far the row is scrolled to put the anchor under the center line, held inside the row. */
export function getAnchorScroll(row: Row, viewport: Size, anchor: Anchor): number | undefined {
  const begins = row.starts[anchor.page]
  const width = row.widths[anchor.page]
  if (begins === undefined || width === undefined) return undefined
  const target = Math.round(begins + anchor.fraction * width - viewport.width / 2)
  return Math.min(Math.max(target, 0), Math.max(row.length - viewport.width, 0))
}

/** The share of the page height that is above the center line of the viewport. */
export function getAcrossShare(row: Row, viewport: Size, offset: number): number {
  return row.height <= 0 ? 0.5 : (offset + viewport.height / 2 - GAP) / row.height
}

/** How far the row is scrolled across to put that share of the page height under the center line. */
export function getAcrossScroll(row: Row, viewport: Size, share: number): number {
  const target = Math.round(GAP + share * row.height - viewport.height / 2)
  return Math.min(Math.max(target, 0), Math.max(row.height + 2 * GAP - viewport.height, 0))
}
