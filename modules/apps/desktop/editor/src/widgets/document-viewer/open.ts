/**
 * Reader state and controls for an open document.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../../shared/core'
import { useDocumentNavigation } from './navigation'
import { useDocumentViewport } from './viewport'
import { useDocumentHighlights } from './highlights'
import type {
  DocumentLayout,
  Documents,
  Page,
  PageHighlight,
  Rect,
} from './types'

export type {
  DocumentLayout,
  Documents,
  Page,
  PageHighlight,
  Rect,
}
export { useDocumentNavigation } from './navigation'
export { useDocumentViewport } from './viewport'
export { useDocumentHighlights } from './highlights'

/** What one open document holds. */
export type DocumentReaderState = ReturnType<typeof useDocumentReader>

export function useDocumentReader(documents: Documents, path: string) {
  const pages = shallowRef<readonly Page[]>([])
  const seen = ref('')
  const error = ref('')
  let open = true

  const navigation = useDocumentNavigation(pages)
  const viewport = useDocumentViewport(documents, path, pages, navigation.at, seen, () => open)

  const loadLayout = async () => {
    try {
      const layout = await documents.getDocumentLayout(path)
      if (!open) return
      pages.value = layout.pages
      seen.value = layout.fingerprint
    } catch (thrown) {
      if (!open) return
      error.value = formatErrorMessage(thrown)
    }
  }

  const layoutPromise = loadLayout()

  const go = async (page: number) => {
    await layoutPromise
    if (!open) return
    await navigation.go(page)
  }

  const highlights = useDocumentHighlights(
    documents,
    path,
    navigation.at,
    error,
    go,
    () => open,
  )

  const next = async () => go(navigation.at.value + 1)
  const back = async () => go(navigation.at.value - 1)

  const reach = async (...spans: readonly Span[]) => {
    await layoutPromise
    if (!open) return
    await highlights.reach(...spans)
  }

  const close = () => {
    open = false
    pages.value = []
  }

  return {
    path,
    pages,
    at: navigation.at,
    picture: viewport.picture,
    getPageImageUrl: viewport.getPageImageUrl,
    pictureOf: viewport.pictureOf,
    highlighted: highlights.highlighted,
    highlightedOn: highlights.highlightedOn,
    also: highlights.also,
    alsoOn: highlights.alsoOn,
    error,
    go,
    next,
    back,
    widen: viewport.widen,
    highlight: highlights.highlight,
    reach,
    close,
  }
}
