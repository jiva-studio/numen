/** What a press on a row comes to: the rows selected, and the row a reach is measured from. */

import type { RowId, ShownRow } from './row'

/**
 * What a press means for the selection: a row joining it or leaving it, or the
 * rows from the anchor reaching this one. A press modified by neither makes the
 * row the whole selection.
 */
export interface Press {
  readonly joining: boolean
  readonly reaching: boolean
}

/** A press with nothing held down. */
export const PLAIN: Press = { joining: false, reaching: false }

/** What a press comes to: the selection, and the row a reach is measured from. */
export interface RowSelection {
  readonly rows: readonly RowId[]
  readonly anchor: RowId | null
}

/**
 * The rows from one to another in the order they are drawn, both ends among
 * them, whichever way round the two stand. A range measured from a row that is
 * not drawn is the row it reaches, alone.
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

  const [start, end] = first <= last ? [first, last] : [last, first]
  return visibleRows.slice(start, end + 1).map((row) => row.id)
}

/** The rows of a set, in the order they are drawn. */
const inOrder = (visibleRows: readonly ShownRow[], ids: ReadonlySet<RowId>): readonly RowId[] =>
  visibleRows.filter((row) => ids.has(row.id)).map((row) => row.id)

/**
 * What a press on a row makes the selection.
 *
 * Reaching takes the rows from the anchor to this one and leaves the anchor
 * where it stands; joining takes the row in or out and puts the anchor on it.
 * The rows come back in the order they are drawn, and a row that is not drawn
 * is among none of them.
 */
export function resolveSelection(
  visibleRows: readonly ShownRow[],
  selection: readonly RowId[],
  anchor: RowId | null,
  row: RowId,
  press: Press,
): RowSelection {
  if (press.reaching) {
    return { rows: getRowsBetween(visibleRows, anchor ?? row, row), anchor: anchor ?? row }
  }

  if (press.joining) {
    const held = new Set(selection)
    if (held.has(row)) held.delete(row)
    else held.add(row)
    return { rows: inOrder(visibleRows, held), anchor: row }
  }

  return { rows: [row], anchor: row }
}

/** Every row that is drawn, with the anchor left where it stands. */
export const everyRow = (visibleRows: readonly ShownRow[], anchor: RowId | null): RowSelection => ({
  rows: visibleRows.map((row) => row.id),
  anchor: anchor ?? visibleRows[0]?.id ?? null,
})

/** Whether two selections hold the same rows in the same order. */
export const isSameSelection = (rows: readonly RowId[], others: readonly RowId[]): boolean =>
  rows.length === others.length && rows.every((row, at) => row === others[at])
