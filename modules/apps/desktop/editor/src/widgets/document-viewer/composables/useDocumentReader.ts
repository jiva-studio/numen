/**
 * Reader state and controls for an open document.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../../../shared/core'
import { useDocumentNavigation } from './useDocumentNavigation'
import { useDocumentViewport } from './useDocumentViewport'
import { useDocumentHighlights } from './useDocumentHighlights'
import type {
  DocumentLayout,
  Documents,
  Page,
  PageHighlight,
  Rect,
} from '../types'

export type {
  DocumentLayout,
  Documents,
  Page,
  PageHighlight,
  Rect,
}
export { useDocumentNavigation } from './useDocumentNavigation'
export { useDocumentViewport } from './useDocumentViewport'
export { useDocumentHighlights } from './useDocumentHighlights'

/** What one open document holds. */
export type DocumentReaderState = ReturnType<typeof useDocumentReader>

export function useDocumentReader(documents: Documents, path: string) {
  const pages = shallowRef<readonly Page[]>([])
  const fingerprint = ref('')
  const error = ref('')
  let open = true

  const navigation = useDocumentNavigation(pages)
  const viewport = useDocumentViewport(
    documents,
    path,
    pages,
    navigation.pageNumber,
    fingerprint,
    () => open,
  )

  const loadLayout = async () => {
    try {
      const layout = await documents.getDocumentLayout(path)
      if (!open) return
      pages.value = layout.pages
      fingerprint.value = layout.fingerprint
    } catch (thrown) {
      if (!open) return
      error.value = formatErrorMessage(thrown)
    }
  }

  const layoutPromise = loadLayout()

  const goToPage = async (page: number) => {
    await layoutPromise
    if (!open) return
    await navigation.goToPage(page)
  }

  const highlights = useDocumentHighlights(
    documents,
    path,
    navigation.pageNumber,
    error,
    goToPage,
    () => open,
  )

  const nextPage = async () => goToPage(navigation.pageNumber.value + 1)
  const prevPage = async () => goToPage(navigation.pageNumber.value - 1)

  const focusSpans = async (...spans: readonly Span[]) => {
    await layoutPromise
    if (!open) return
    await highlights.focusSpans(...spans)
  }

  const close = () => {
    open = false
    pages.value = []
  }

  return {
    path,
    pages,
    pageNumber: navigation.pageNumber,
    picture: viewport.picture,
    getPageImageUrl: viewport.getPageImageUrl,
    highlighted: highlights.highlighted,
    highlightedOn: highlights.highlightedOn,
    also: highlights.also,
    alsoOn: highlights.alsoOn,
    error,
    goToPage,
    nextPage,
    prevPage,
    widen: viewport.widen,
    applyHighlights: highlights.applyHighlights,
    focusSpans,
    close,
  }
}
