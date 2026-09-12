/**
 * Which rows the selection stands on, and where a reach is measured from.
 *
 * The rows themselves are the caller's: a press is worked out here and the
 * selection it came to is handed back.
 */
import { computed, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { RowId, ShownRow } from '../lib/row'
import { everyRow, resolveSelection, sameRows, type Press, type RowSelection } from '../lib/select'

export interface RowSelectionState {
  /** The rows selected, for asking one row at a time. */
  readonly picked: ComputedRef<ReadonlySet<RowId>>
  /** Whether the press being made has said what the selection is already. */
  readonly said: Ref<boolean>
  /** What a press on a row makes the selection, said and handed back. */
  readonly selectRow: (row: RowId, how: Press) => readonly RowId[]
  /** Every drawn row selected. */
  readonly selectEveryRow: () => readonly RowId[]
}

export function useRowSelection(
  shown: () => readonly ShownRow[],
  selected: () => readonly RowId[],
  /** The rows the selection now stands on, said only where they have changed. */
  select: (rows: readonly RowId[]) => void,
): RowSelectionState {
  const picked = computed(() => new Set(selected()))

  /** The row a reach is measured from, where a plain or joining press last landed. */
  const anchor = shallowRef<RowId | null>(null)

  const said = shallowRef(false)

  /** A selection a press came to, said, and the anchor put where it names. */
  const apply = (pressed: RowSelection): readonly RowId[] => {
    anchor.value = pressed.anchor
    if (!sameRows(pressed.rows, selected())) select(pressed.rows)
    return pressed.rows
  }

  const selectRow = (row: RowId, how: Press): readonly RowId[] =>
    apply(resolveSelection(shown(), selected(), anchor.value, row, how))

  const selectEveryRow = (): readonly RowId[] => apply(everyRow(shown(), anchor.value))

  return { picked, said, selectRow, selectEveryRow }
}
