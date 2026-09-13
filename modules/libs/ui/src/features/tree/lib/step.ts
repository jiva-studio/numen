/** The keys that move the keyboard about a tree, and where each of them takes it. */

import type { RowId, ShownRow } from './row'

/** The keys that move the keyboard about a tree, declared once. */
export const TREE_KEYS = ['ArrowDown', 'ArrowUp', 'ArrowRight', 'ArrowLeft', 'Home', 'End'] as const

export type TreeKey = (typeof TREE_KEYS)[number]

export const isTreeKey = (key: string): key is TreeKey =>
  (TREE_KEYS as readonly string[]).includes(key)

/** What a key comes to: a row to turn on the way, and where the keyboard lands. */
export interface Step {
  /** Where the keyboard is afterwards. Null while nothing is drawn. */
  readonly at: RowId | null
  /** The row to open, or to close, before it lands. */
  readonly turn: { readonly row: RowId; readonly open: boolean } | null
}

/** What one key does, told the rows, where the keyboard stands in them, and the row it stands on. */
type Move = (visibleRows: readonly ShownRow[], at: number, here: ShownRow) => Step

const MOVES: Record<TreeKey, Move> = {
  Home: stepToFirst,
  End: stepToLast,
  ArrowDown: stepDown,
  ArrowUp: stepUp,
  ArrowRight: stepIn,
  ArrowLeft: stepOut,
}

/**
 * Where a key takes the keyboard.
 *
 * From no row at all every key lands on the first.
 */
export function stepTo(visibleRows: readonly ShownRow[], from: RowId | null, key: TreeKey): Step {
  const at = from === null ? -1 : visibleRows.findIndex((row) => row.id === from)
  const here = at === -1 ? undefined : visibleRows[at]
  if (!here) return stepToFirst(visibleRows)

  return MOVES[key](visibleRows, at, here)
}

function stepToFirst(visibleRows: readonly ShownRow[]): Step {
  return { at: visibleRows[0]?.id ?? null, turn: null }
}

function stepToLast(visibleRows: readonly ShownRow[]): Step {
  return { at: visibleRows[visibleRows.length - 1]?.id ?? null, turn: null }
}

function stepDown(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {
  return { at: visibleRows[at + 1]?.id ?? here.id, turn: null }
}

function stepUp(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {
  return { at: visibleRows[at - 1]?.id ?? here.id, turn: null }
}

/** A closed row opens, an open one is descended into, and a leaf stays. */
function stepIn(visibleRows: readonly ShownRow[], at: number, here: ShownRow): Step {
  if (here.holds && !here.open) return { at: here.id, turn: { row: here.id, open: true } }
  if (here.open && here.holding) return stepDown(visibleRows, at, here)
  return { at: here.id, turn: null }
}

/** An open row closes, and a closed one climbs to its holder. */
function stepOut(_visibleRows: readonly ShownRow[], _at: number, here: ShownRow): Step {
  if (here.open) return { at: here.id, turn: { row: here.id, open: false } }
  return { at: here.parent ?? here.id, turn: null }
}
