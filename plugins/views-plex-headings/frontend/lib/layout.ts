import type { Measures } from './measures'
import type { NodeView } from '../types'

/** What the list needs to be drawn under a node: how many rows stand at once and where the node is. */
export interface ListLayout {
  readonly rowHeight: number
  readonly pad: number
  readonly margin: number
  readonly visibleCount: number
  readonly height: number
  readonly nodeX: number
  readonly nodeWidth: number
  readonly viewportWidth: number
}

export interface ListLayoutRequest {
  readonly count: number
  readonly limit: number
  readonly node: Pick<NodeView, 'x' | 'y' | 'width' | 'height' | 'viewport'>
  readonly measures: Measures
}

/**
 * The layout of a list of `count` rows hung from the foot of a node, of which at most `limit` stand at once and
 * only as many as the depth below the node holds. None for a list with no rows or where no row fits.
 */
export function getListLayout(request: ListLayoutRequest): ListLayout | null {
  const { count, limit, node, measures } = request
  const { rowHeight, pad, margin } = measures
  if (count === 0) return null

  const depth = node.viewport.height / 2 - margin - (node.y + node.height / 2)
  const rows = Math.min(limit, Math.floor((depth - 2 * pad) / rowHeight))
  if (rows < 1) return null

  const visibleCount = Math.min(rows, count)
  return {
    rowHeight,
    pad,
    margin,
    visibleCount,
    height: visibleCount * rowHeight + 2 * pad,
    nodeX: node.x,
    nodeWidth: node.width,
    viewportWidth: node.viewport.width,
  }
}

/** The furthest the window on the rows may be scrolled to, counted in rows. */
export const getFurthestOffset = (count: number, layout: ListLayout): number =>
  Math.max(0, count - layout.visibleCount)

/** Where the list begins, counted from the start of the node and in the units of the surface: centered on the node, and kept inside the window. */
export function getListStart(layout: ListLayout, width: number): number {
  const furthest = layout.viewportWidth / 2 - layout.margin - width / 2
  const middle = Math.min(Math.max(layout.nodeX, -furthest), furthest)
  return (layout.nodeWidth - width) / 2 + (middle - layout.nodeX)
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
