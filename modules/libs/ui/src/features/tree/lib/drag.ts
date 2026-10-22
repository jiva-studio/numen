/** The rows a press takes with it, and what follows the pointer while they are on their way. */

import type { Position } from '@/shared/lib/math/geometry'
import type { RowId, ShownRow } from './row'

/**
 * and the row alone at its current position outside.
 */
export const getDraggedRows = (selection: readonly RowId[], row: RowId): readonly RowId[] =>
  selection.includes(row) ? selection : [row]

/** What is drawn at the pointer while rows are dragged. */
export interface DragLabel {
  /** The name of the one row dragged, or how many there are. */
  readonly label: string
  /** Where the pointer is, which is where it is drawn. */
  readonly at: Position
}

export interface DragLabelOptions {
  readonly at: Position
  readonly getCountWords: (rows: number) => string
}

/**
 * What follows the pointer while rows are dragged, and nothing while none are.
 * One row is said by its name; several are said by how many.
 */
export function dragLabel(
  visibleRows: readonly ShownRow[],
  rows: readonly RowId[],
  options: DragLabelOptions,
): DragLabel | null {
  const first = rows[0]
  if (first === undefined) return null

  const label =
    rows.length === 1
      ? (visibleRows.find((row) => row.id === first)?.name ?? first)
      : options.getCountWords(rows.length)
  return { label, at: options.at }
}
