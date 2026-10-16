/**
 * Tab state for document tabs.
 */
import { shallowRef } from 'vue'
import type { DocumentReaderState } from './useDocumentReader'
import type { PageHandle } from '../types'

export type { PageHandle }

/** What one document tab holds. */
export type DocumentTabState = ReturnType<typeof useDocumentTab>

export function useDocumentTab(read: DocumentReaderState) {
  /** The page of this document, for as long as its tab is drawn. */
  const page = shallowRef<PageHandle | null>(null)

  const setPageHandle = (handle: unknown) => {
    page.value = (handle as PageHandle | null) ?? null
  }

  const measure = () => page.value?.measure()
  const handleKeyPress = (event: KeyboardEvent) => page.value?.handleKey(event) ?? false
  const focusTab = () => page.value?.focusPages()

  return {
    ...read,
    setPageHandle,
    measure,
    handleKeyPress,
    focusTab,
  }
}
