/** The keys that move the keyboard about a tree, and where each of them takes it. */

import type { RowId, ShownRow } from './row'

/** The keys that move the keyboard about a tree, declared once. */
export const TREE_KEYS = ['ArrowDown', 'ArrowUp', 'ArrowRight', 'ArrowLeft', 'Home', 'End'] as const

export type TreeKey = (typeof TREE_KEYS)[number]

export const isTreeKey = (key: string): key is TreeKey =>
  (TREE_KEYS as readonly string[]).includes(key)

/** What a key evaluates to: a row to turn on the way, and where the keyboard lands. */
export interface Step {
  readonly at: RowId | null
  readonly turn: { readonly row: RowId; readonly isOpen: boolean } | null
}

type Move = (visibleRows: readonly ShownRow[], pos: number, here: ShownRow) => Step

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
  const index = visibleRows.findIndex((row) => row.id === from)
  const here = visibleRows[index]
  if (!here) return stepToFirst(visibleRows)

  return MOVES[key](visibleRows, index, here)
}

function stepToFirst(visibleRows: readonly ShownRow[]): Step {
  return { at: visibleRows[0]?.id ?? null, turn: null }
}

function stepToLast(visibleRows: readonly ShownRow[]): Step {
  return { at: visibleRows.at(-1)?.id ?? null, turn: null }
}

function stepDown(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {
  return { at: visibleRows[pos + 1]?.id ?? here.id, turn: null }
}

function stepUp(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {
  return { at: visibleRows[pos - 1]?.id ?? here.id, turn: null }
}

function stepIn(visibleRows: readonly ShownRow[], pos: number, here: ShownRow): Step {
  if (here.hasChildren && !here.isOpen) return { at: here.id, turn: { row: here.id, isOpen: true } }
  if (here.isOpen && here.hasRows) return stepDown(visibleRows, pos, here)
  return { at: here.id, turn: null }
}

function stepOut(_visibleRows: readonly ShownRow[], _pos: number, here: ShownRow): Step {
  if (here.isOpen) return { at: here.id, turn: { row: here.id, isOpen: false } }
  return { at: here.parent ?? here.id, turn: null }
}
