/**
 * The rows as they are drawn, and which one of them the keyboard reaches.
 *
 * A row is held under the identity it stands for, so the keyboard can be put on
 * a row that has only just arrived.
 */
import { computed, nextTick, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { RowId, ShownRow } from '../lib/row'

export interface DrawnRowsState {
  /** The row the keyboard was last on. */
  readonly here: Ref<RowId | null>
  /**
   * The one row the tab key reaches: where the keyboard was left, else the
   * first row of the selection, else the first row of all.
   */
  readonly tabbed: ComputedRef<RowId | null>
  /** A row as it is drawn, held under the row it stands for. */
  readonly setRowElement: (row: RowId, element: unknown) => void
  readonly getRowElement: (row: RowId) => HTMLElement | null
  /** The keyboard onto a row, once the rows it moved among are drawn. */
  readonly focusRow: (row: RowId | null) => Promise<void>
}

export function useDrawnRows(
  getVisibleRows: () => readonly ShownRow[],
  getSelection: () => readonly RowId[],
): DrawnRowsState {
  /** The rows as they are drawn, each under the row it stands for. */
  const drawn = new Map<RowId, HTMLElement>()

  const here = shallowRef<RowId | null>(null)

  const tabbed = computed<RowId | null>(() => {
    const getDrawnRow = (row: RowId | null | undefined) =>
      row != null && getVisibleRows().some((each) => each.id === row) ? row : null
    return (
      getDrawnRow(here.value) ?? getDrawnRow(getSelection()[0]) ?? getVisibleRows()[0]?.id ?? null
    )
  })

  const setRowElement = (row: RowId, element: unknown): void => {
    const found = (element as { $el?: unknown } | null)?.$el
    if (found) drawn.set(row, found as HTMLElement)
    else drawn.delete(row)
  }

  const getRowElement = (row: RowId): HTMLElement | null => drawn.get(row) ?? null

  const focusRow = async (row: RowId | null): Promise<void> => {
    if (row === null) return
    here.value = row
    await nextTick()
    getRowElement(row)?.focus()
  }

  return { here, tabbed, setRowElement, getRowElement, focusRow }
}
