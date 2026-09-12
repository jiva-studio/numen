/** The rows a press takes with it, and what follows the pointer while they are on their way. */

import type { Position } from '@/shared/lib/geometry'
import type { RowId, ShownRow } from './row'

/**
 * The rows a press on a row drags: the selection, where the row stands in it,
 * and the row alone where it stands outside.
 */
export const dragged = (selected: readonly RowId[], row: RowId): readonly RowId[] =>
  selected.includes(row) ? selected : [row]

/** What is drawn at the pointer while rows are dragged. */
export interface DragLabel {
  /** The name of the one row dragged, or how many there are. */
  readonly says: string
  /** Where the pointer is, which is where it is drawn. */
  readonly at: Position
}

/**
 * What follows the pointer while rows are dragged, and nothing while none are.
 * One row is said by its name; several are said by how many.
 */
export function dragLabel(
  shown: readonly ShownRow[],
  rows: readonly RowId[],
  at: Position,
  counted: (rows: number) => string,
): DragLabel | null {
  const first = rows[0]
  if (first === undefined) return null

  const says =
    rows.length === 1 ? (shown.find((row) => row.id === first)?.name ?? first) : counted(rows.length)
  return { says, at }
}
