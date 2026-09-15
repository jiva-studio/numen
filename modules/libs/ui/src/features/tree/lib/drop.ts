/** Where rows let go at a height land, and which landings are refused. */

import type { Row, RowId, ShownRow } from './row'

/**
 * Where held rows would land: inside a row, or above one. Into no row at all
 * is the top level.
 */
export type RowLanding = { readonly into: RowId | null } | { readonly before: RowId }

/** The row a landing names, and nothing for the top level. */
const getLandingRow = (at: RowLanding): RowId | null => ('into' in at ? at.into : at.before)

/**
 * What letting go at a height comes to.
 *
 * A row that holds is read in three bands: the middle half means into it, and
 * the quarter at either end means between. Every other row is halved, and each
 * half means between. Past the last row is the top level, which is what the
 * tree's own empty area comes to. A landing naming one of the rows being
 * dragged moves nothing and answers nothing.
 */
export function landing(
  visibleRows: readonly ShownRow[],
  dragIds: readonly RowId[],
  y: number,
  height: number,
): RowLanding | null {
  const found = bandAt(visibleRows, y, height)
  if (!found) return null

  const on = getLandingRow(found)
  return on !== null && dragIds.includes(on) ? null : found
}

/** The band a height falls in, as a landing. */
function bandAt(visibleRows: readonly ShownRow[], y: number, height: number): RowLanding | null {
  if (height <= 0 || y < 0) return null

  const at = Math.floor(y / height)
  const row = visibleRows[at]
  if (!row) return { into: null }

  const band = y - at * height
  const edge = row.holds ? height / 4 : height / 2

  if (band < edge) return { before: row.id }
  if (band < height - edge) return { into: row.id }

  const next = visibleRows[at + 1]
  return next ? { before: next.id } : { into: null }
}

/** The row a landing puts what is held inside. Null at the top level. */
export const holderOf = (visibleRows: readonly ShownRow[], at: RowLanding): RowId | null =>
  'into' in at ? at.into : (visibleRows.find((row) => row.id === at.before)?.parent ?? null)

/**
 * Rows cannot land in one of themselves, nor in anything one of them holds.
 * Everything else is allowed, and the top level refuses nothing.
 */
export function isRefused(
  rows: readonly Row[],
  dragIds: readonly RowId[],
  into: RowId | null,
): boolean {
  if (into === null) return false

  const lifted = new Set(dragIds)
  if (lifted.has(into)) return true

  const isInsideLifted = (children: readonly Row[], within: boolean): boolean =>
    children.some(
      (row) =>
        (within && row.id === into) || isInsideLifted(row.rows ?? [], within || lifted.has(row.id)),
    )

  return isInsideLifted(rows, false)
}
