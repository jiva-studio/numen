/**
 * Reader state and controls for an open document.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { TextRange } from '../types'
import { useDocumentNavigation } from './useDocumentNavigation'
import { useDocumentViewport } from './useDocumentViewport'
import { useDocumentHighlights } from './useDocumentHighlights'
import type { DocumentLayout, Documents, Page, PageHighlight, Rect } from '../types'

export type { DocumentLayout, Documents, Page, PageHighlight, Rect }
export { useDocumentNavigation } from './useDocumentNavigation'
export { useDocumentViewport } from './useDocumentViewport'
export { useDocumentHighlights } from './useDocumentHighlights'

/** What one open document holds. */
export type DocumentReaderState = ReturnType<typeof useDocumentReader>

function createDocumentReaderNavigation(
  getLayoutPromise: () => Promise<void>,
  navigation: ReturnType<typeof useDocumentNavigation>,
  highlights: ReturnType<typeof useDocumentHighlights>,
) {
  const goToPage = async (page: number) => {
    await getLayoutPromise()
    await navigation.goToPage(page)
  }

  const goToNextPage = async () => goToPage(navigation.pageNumber.value + 1)
  const goToPreviousPage = async () => goToPage(navigation.pageNumber.value - 1)

  const focusSpans = async (...spans: readonly TextRange[]) => {
    await getLayoutPromise()
    await highlights.focusSpans(...spans)
  }

  return { goToPage, goToNextPage, goToPreviousPage, focusSpans }
}

export function useDocumentReader(documents: Documents, path: string) {
  const pages = shallowRef<readonly Page[]>([])
  const stat = ref<string>()
  const error = ref('')
  const isLoading = ref(true)
  let open = true

  const navigation = useDocumentNavigation(pages)
  const viewport = useDocumentViewport(documents, path, {
    pages,
    pageNumber: navigation.pageNumber,
    stat,
    isOpen: () => open,
  })

  const loadLayout = async () => {
    try {
      const layout = await documents.getDocumentLayout(path)
      if (!open) return
      pages.value = layout.pages
      stat.value = layout.stat
    } catch (caughtError) {
      if (!open) return
      error.value = formatErrorMessage(caughtError)
    } finally {
      isLoading.value = false
    }
  }

  const layoutPromise = loadLayout()

  const highlights = useDocumentHighlights(documents, path, {
    pageNumber: navigation.pageNumber,
    error,
    onGoToPage: (page) => nav.goToPage(page),
    isOpen: () => open,
  })

  // Stryker disable next-line ArrowFunction: Navigation awaits the layout loading promise.
  const nav = createDocumentReaderNavigation(() => layoutPromise, navigation, highlights)

  const close = () => {
    open = false
    pages.value = []
  }

  const reader = {
    path,
    pages,
    pageNumber: navigation.pageNumber,
    picture: viewport.picture,
    getPageImageUrl: viewport.getPageImageUrl,
    highlighted: highlights.highlighted,
    getHighlightsOn: highlights.getHighlightsOn,
    otherHighlighted: highlights.otherHighlighted,
    getOtherHighlightsOn: highlights.getOtherHighlightsOn,
    error,
    isLoading,
    goToPage: nav.goToPage,
    goToNextPage: nav.goToNextPage,
    goToPreviousPage: nav.goToPreviousPage,
    widen: viewport.widen,
    applyHighlights: highlights.applyHighlights,
    focusSpans: nav.focusSpans,
    close,
  }
  return reader
}
