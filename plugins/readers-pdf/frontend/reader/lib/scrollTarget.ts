import type { Size } from '@numen/ui'
import { findPageScroll } from './pageScroll'
import type { Row } from './strip'

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
