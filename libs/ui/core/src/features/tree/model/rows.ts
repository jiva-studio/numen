/**
 * The rows as rendered, and the row that keyboard focus reaches.
 *
 * A row element is stored under the identity of the row it represents, so
 * keyboard focus can be set on a row that was just added.
 */
import { computed, nextTick, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { RowId, ShownRow } from '../lib/row'

export interface DrawnRowsState {
  /** The row the keyboard was last on. */
  readonly here: Ref<RowId | null>
  /**
   * The single row that the tab key reaches: the last keyboard row, else the
   * first selected row, else the first row.
   */
  readonly tabbed: ComputedRef<RowId | null>
  /** Registers the element rendered for a row. */
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
  /** The rendered elements, keyed by row identity. */
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
