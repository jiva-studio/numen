import type { TextMeasures } from './measures'
import type { Box, Size } from '../types'

/** What the list needs to be drawn under a node: how many rows stand at once and where the node is on the screen. */
export interface ListLayout {
  readonly rowHeight: number
  readonly margin: number
  readonly visibleCount: number
  readonly height: number
  readonly top: number
  readonly nodeLeft: number
  readonly nodeWidth: number
  readonly roomWidth: number
}

export interface ListLayoutRequest {
  readonly count: number
  readonly limit: number
  readonly node: Box
  readonly room: Size
  readonly measures: TextMeasures
}

/**
 * The layout of a list of `count` rows hung from the foot of a node, of which at most `limit` stand at once and
 * only as many as the depth below the node holds in the `room` of the window. The node is the box it is drawn in on the screen. None for a list with no rows or where no row fits.
 */
export function getListLayout(request: ListLayoutRequest): ListLayout | null {
  const { count, limit, node, room, measures } = request
  const { rowHeight, margin } = measures
  if (count === 0) return null

  const depth = room.height - margin - (node.y + node.height)
  const rows = Math.min(limit, Math.floor(depth / rowHeight))
  if (rows < 1) return null

  const visibleCount = Math.min(rows, count)
  return {
    rowHeight,
    margin,
    visibleCount,
    height: visibleCount * rowHeight,
    top: node.y + node.height,
    nodeLeft: node.x,
    nodeWidth: node.width,
    roomWidth: room.width,
  }
}

/** The furthest the window on the rows may be scrolled to, counted in rows. */
export const getFurthestOffset = (count: number, layout: ListLayout): number =>
  Math.max(0, count - layout.visibleCount)

/** Where the list begins, counted from the left of the window: centered on the node, and kept inside the margins of the window. */
export function getListStart(layout: ListLayout, width: number): number {
  const centered = layout.nodeLeft + (layout.nodeWidth - width) / 2
  return Math.max(layout.margin, Math.min(centered, layout.roomWidth - layout.margin - width))
}

/** The wheel delta units, numbered as the browser's `deltaMode` numbers them. */
const LINES = 1
const PAGES = 2

function getStride(layout: ListLayout, mode: number): number {
  if (mode === LINES) return layout.rowHeight
  if (mode === PAGES) return layout.visibleCount * layout.rowHeight
  return 1
}

/**
 * How far a wheel scrolls the window, in whole rows, and the remainder. The caller carries the remainder into
 * the next wheel event, so a trackpad that reports a few pixels at a time adds up to whole rows. Deltas in lines
 * or pages are turned into pixels first, because only pixels can be compared with a row height.
 */
export function scrollBy(
  layout: ListLayout,
  wheel: { readonly delta: number; readonly mode: number },
  leftover: number,
): { by: number; left: number } {
  const pixels = leftover + wheel.delta * getStride(layout, wheel.mode)
  const rows = Math.trunc(pixels / layout.rowHeight)
  return { by: rows, left: pixels - rows * layout.rowHeight }
}
