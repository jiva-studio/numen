/** Carries the places and ranges the window sets on a reader tab to the document, and the page the tab shows back to the window. */
import { watch } from 'vue'
import type { ReaderTabState, Selector } from '@numen/plugin-sdk'
import type { DocumentTabState } from './useDocumentTab'

export function useReaderTabSync(
  getReaderTab: () => Partial<ReaderTabState> | undefined,
  getDocument: () => DocumentTabState | null,
): void {
  // The selector is a search hit or the place saved before a restart; `page` counts from 1 (RFC 8118).
  watch(
    () => getReaderTab()?.selector,
    (selector: Selector | null | undefined) => {
      const tab = getDocument()
      if (!tab || typeof selector?.page !== 'number') return
      const page = Math.max(0, selector.page - 1)
      const boxes = selector.boxes as
        readonly (readonly [number, number, number, number])[] | undefined
      if (Array.isArray(boxes) && boxes.length > 0) {
        const rects = boxes.map(([minX, minY, maxX, maxY]) => ({
          minX,
          minY,
          maxX,
          maxY,
        }))
        tab.applyHighlights([[{ page, rects }]]).catch(tab.reportFailure)
        return
      }
      tab.goToPage(page).catch(tab.reportFailure)
    },
    { immediate: true },
  )

  // The window sets a new array on each request, so the same ranges are highlighted again.
  watch(
    () => getReaderTab()?.ranges,
    (ranges) => {
      const tab = getDocument()
      if (!tab || !ranges || ranges.length === 0) return
      tab.focusSpans(...ranges).catch(tab.reportFailure)
    },
    { immediate: true },
  )

  watch(
    () => getDocument()?.pageNumber.value,
    (page) => {
      const pageCount = getDocument()?.pages.value.length ?? 0
      if (typeof page === 'number' && pageCount > 0 && !getDocument()?.isLoading.value) {
        getReaderTab()?.savePlace?.({ page: page + 1, pageCount })
      }
    },
  )

  // The place is reported as soon as the document has loaded, the page it opens at included, and again
  // on each later page.
  watch(
    () =>
      [
        getDocument()?.pageNumber.value,
        getDocument()?.pages.value.length ?? 0,
        getDocument()?.isLoading.value,
      ] as const,
    ([page, pageCount, isLoading]) => {
      if (typeof page === 'number' && pageCount > 0 && !isLoading) {
        getReaderTab()?.reportPlace?.({ page: page + 1, pageCount })
      }
    },
    { immediate: true },
  )
}
