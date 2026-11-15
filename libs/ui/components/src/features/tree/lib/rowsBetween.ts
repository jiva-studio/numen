import type { RowId, ShownRow } from './row'

/**
 * The rows between two rows in display order, both ends included, in either
 * direction. A range anchored at a row that is not displayed is that row
 * alone.
 */
export function getRowsBetween(
  visibleRows: readonly ShownRow[],
  from: RowId,
  to: RowId,
): readonly RowId[] {
  const last = visibleRows.findIndex((row) => row.id === to)
  if (last === -1) return []

  const first = visibleRows.findIndex((row) => row.id === from)
  if (first === -1) return [to]

  return visibleRows.slice(Math.min(first, last), Math.max(first, last) + 1).map((row) => row.id)
}
