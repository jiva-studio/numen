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
  /** Whether the current press has already applied the selection. */
  readonly hasApplied: Ref<boolean>
  /** Updates the selection for a pressed row and returns the new selection. */
  readonly selectRow: (row: RowId, how: Press) => readonly RowId[]
  /** Every drawn row selected. */
  readonly selectEveryRow: () => readonly RowId[]
}

export function useRowSelection(
  getVisibleRows: () => readonly ShownRow[],
  getSelection: () => readonly RowId[],
  /** The callback invoked when selected rows change. */
  select: (rows: readonly RowId[]) => void,
): RowSelectionState {
  const picked = computed(() => new Set(getSelection()))

  /** The row a reach is measured from, where a plain or joining press last landed. */
  const anchor = shallowRef<RowId | null>(null)

  const hasApplied = shallowRef(false)

  /** Applies a resolved row selection and updates the anchor. */
  const apply = (selection: RowSelection): readonly RowId[] => {
    anchor.value = selection.anchor
    if (!isSameSelection(selection.rows, getSelection())) select(selection.rows)
    return selection.rows
  }

  const selectRow = (row: RowId, how: Press): readonly RowId[] =>
    apply(resolveSelection(getVisibleRows(), getSelection(), anchor.value, row, how))

  const selectEveryRow = (): readonly RowId[] => apply(everyRow(getVisibleRows(), anchor.value))

  return { picked, hasApplied, selectRow, selectEveryRow }
}
