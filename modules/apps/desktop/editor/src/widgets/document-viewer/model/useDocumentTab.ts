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

  const setPageHandle = (drawn: unknown) => {
    page.value = (drawn as PageHandle | null) ?? null
  }

  const measure = () => page.value?.measure()

  return {
    ...read,
    setPageHandle,
    measure,
  }
}
