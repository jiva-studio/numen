/**
 * The rows as rendered, and the row that receives keyboard focus.
 *
 * A row element is stored under the identity of the row it represents, so
 * keyboard focus can be set on a row added in the same update.
 */
import { computed, nextTick, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { RefTarget } from '@/shared/model/refs'
import type { RowId, ShownRow } from '../lib/row'

export interface DrawnRowsState {
  readonly here: Ref<RowId | null>
  readonly tabbed: ComputedRef<RowId | null>
  readonly setRowElement: (row: RowId, element: RefTarget) => void
  readonly getRowElement: (row: RowId) => HTMLElement | null
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

  const setRowElement = (row: RowId, element: RefTarget): void => {
    const found: HTMLElement | null = element && '$el' in element ? element.$el : null
    // Stryker disable next-line ConditionalExpression: deleting absent elements prevents map retention while getRowElement coalesces undefined to null
    if (found) viewElement.set(row, found)
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
