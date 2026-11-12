import type { Row, RowId, ShownRow } from './row'

/** Drop target for dragged rows: inside a parent row or before a row. */
export type RowDropTarget = { readonly into: RowId | null } | { readonly before: RowId }

const getTargetRow = (target: RowDropTarget): RowId | null =>
  'into' in target ? target.into : target.before

/** Vertical pointer offset and row height for drop target calculation. */
export interface DropMetrics {
  readonly y: number
  readonly height: number
}

/** Returns the row drop target at a vertical offset, or null when the target is one of the dragged rows. */
export function getRowDropTarget(
  visibleRows: readonly ShownRow[],
  dragIds: readonly RowId[],
  metrics: DropMetrics,
): RowDropTarget | null {
  const found = findBand(visibleRows, metrics.y, metrics.height)
  if (!found) return null

  const on = getTargetRow(found)
  return dragIds.some((id) => id === on) ? null : found
}

function findBand(
  visibleRows: readonly ShownRow[],
  y: number,
  height: number,
): RowDropTarget | null {
  if (height <= 0 || y < 0) return null

  const index = Math.floor(y / height)
  const row = visibleRows[index]
  if (!row) return { into: null }

  const band = y - index * height
  const edge = row.hasChildren ? height / 4 : height / 2

  if (band < edge) return { before: row.id }
  if (band < height - edge) return { into: row.id }

  const next = visibleRows[index + 1]
  return next ? { before: next.id } : { into: null }
}

/** Returns the parent row id for a drop target, or null at the root level. */
export const findHolder = (
  visibleRows: readonly ShownRow[],
  target: RowDropTarget,
): RowId | null =>
  'into' in target
    ? target.into
    : (visibleRows.find((row) => row.id === target.before)?.parent ?? null)

/** Returns true when dropping the dragged rows into the target parent row is not allowed. */
export function isRefused(
  rows: readonly Row[],
  dragIds: readonly RowId[],
  into: RowId | null,
): boolean {
  const dragged = new Set<RowId | null>(dragIds)
  if (dragged.has(into)) return true

  const isInsideDragged = (children: readonly Row[], within: boolean): boolean =>
    children.some(
      (row) =>
        (within && row.id === into) ||
        isInsideDragged(row.rows ?? [], within || dragged.has(row.id)),
    )

  return isInsideDragged(rows, false)
}
