import { GAP, type Row } from './strip'

/** How far the row is scrolled to put one page against the left edge. */
export function findPageScroll(row: Row, page: number): number | undefined {
  const begins = row.starts[page]
  return begins === undefined ? undefined : begins - GAP
}
