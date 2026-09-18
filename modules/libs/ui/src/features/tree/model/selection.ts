/**
 * Which rows the selection stands on, and where a reach is measured from.
 *
 * The rows themselves are the caller's: a press is worked out here and the
 * selection it came to is handed back.
 */
import { computed, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { RowId, ShownRow } from '../lib/row'
import {
  everyRow,
  resolveSelection,
  isSameSelection,
  type Press,
  type RowSelection,
} from '../lib/select'

export interface RowSelectionState {
  /** The rows selected, for asking one row at a time. */
  readonly picked: ComputedRef<ReadonlySet<RowId>>
  /** Whether the press being made has said what the selection is already. */
  readonly wasSaid: Ref<boolean>
  /** What a press on a row makes the selection, said and handed back. */
  readonly selectRow: (row: RowId, how: Press) => readonly RowId[]
  /** Every drawn row selected. */
  readonly selectEveryRow: () => readonly RowId[]
}

export function useRowSelection(
  getVisibleRows: () => readonly ShownRow[],
  getSelection: () => readonly RowId[],
  /** The rows the selection now stands on, said only where they have changed. */
  select: (rows: readonly RowId[]) => void,
): RowSelectionState {
  const picked = computed(() => new Set(getSelection()))

  /** The row a reach is measured from, where a plain or joining press last landed. */
  const anchor = shallowRef<RowId | null>(null)

  const wasSaid = shallowRef(false)

  /** A selection a press came to, said, and the anchor put where it names. */
  const apply = (selection: RowSelection): readonly RowId[] => {
    anchor.value = selection.anchor
    if (!isSameSelection(selection.rows, getSelection())) select(selection.rows)
    return selection.rows
  }

  const selectRow = (row: RowId, how: Press): readonly RowId[] =>
    apply(resolveSelection(getVisibleRows(), getSelection(), anchor.value, row, how))

  const selectEveryRow = (): readonly RowId[] => apply(everyRow(getVisibleRows(), anchor.value))

  return { picked, wasSaid, selectRow, selectEveryRow }
}
