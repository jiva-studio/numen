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

/**
 * Where a key takes the keyboard.
 *
 * Down and up move a row and stop at the ends. Right opens a closed row and
 * then descends into it; left closes an open row and then climbs to its
 * holder. Home and End go to the ends, and from no row at all every key lands
 * on the first.
 */
export function stepTo(shown: readonly ShownRow[], from: RowId | null, key: TreeKey): Step {
  const first = shown[0]?.id ?? null
  const at = from === null ? -1 : shown.findIndex((row) => row.id === from)
  const here = at === -1 ? undefined : shown[at]
  if (!here) return { at: first, turn: null }

  const stays: Step = { at: here.id, turn: null }

  switch (key) {
    case 'Home':
      return { at: first, turn: null }
    case 'End':
      return { at: shown[shown.length - 1]?.id ?? null, turn: null }
    case 'ArrowDown':
      return { at: shown[at + 1]?.id ?? here.id, turn: null }
    case 'ArrowUp':
      return { at: shown[at - 1]?.id ?? here.id, turn: null }
    case 'ArrowRight':
      if (here.holds && !here.open) return { at: here.id, turn: { row: here.id, open: true } }
      if (here.open && here.holding) return { at: shown[at + 1]?.id ?? here.id, turn: null }
      return stays
    case 'ArrowLeft':
      if (here.open) return { at: here.id, turn: { row: here.id, open: false } }
      return { at: here.parent ?? here.id, turn: null }
  }
}
