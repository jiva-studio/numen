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
  readonly focusRow: (row: RowId | null, scrollTo?: (index: number) => void) => Promise<void>
}

export function useDrawnRows(
  getVisibleRows: () => readonly ShownRow[],
  getSelection: () => readonly RowId[],
  onScrollToRow?: (index: number) => void,
): DrawnRowsState {
  /** The rows as they are drawn, each under the row it stands for. */
  const viewElement = new Map<RowId, HTMLElement>()

  const here = shallowRef<RowId | null>(null)

  const tabbed = computed<RowId | null>(() => {
    const getDrawnRow = (row: RowId | null | undefined) =>
      getVisibleRows().find((each) => each.id === row)?.id ?? null
    return (
      getDrawnRow(here.value) ?? getDrawnRow(getSelection()[0]) ?? getVisibleRows()[0]?.id ?? null
    )
  })

  const setRowElement = (row: RowId, element: unknown): void => {
    const found = (element as { $el?: unknown } | null)?.$el
    if (found) viewElement.set(row, found as HTMLElement)
    else viewElement.delete(row)
  }

  const getRowElement = (row: RowId): HTMLElement | null => viewElement.get(row) ?? null

  const focusRow = async (row: RowId | null, scrollTo?: (index: number) => void): Promise<void> => {
    if (row === null) return
    here.value = row
    const scroll = scrollTo ?? onScrollToRow
    if (scroll) {
      const index = getVisibleRows().findIndex((each) => each.id === row)
      if (index !== -1) {
        scroll(index)
      }
    }
    await nextTick()
    getRowElement(row)?.focus()
  }

  return { here, tabbed, setRowElement, getRowElement, focusRow }
}
