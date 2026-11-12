/**
 * Tab state for document tabs.
 */
import { shallowRef } from 'vue'
import type { Unvalidated } from '@numen/plugin-sdk'
import type { DocumentReaderState } from './useDocumentReader'
import type { PageHandle } from '../types'

export type { PageHandle }

/** Whether a ref callback received the page view, which exposes the methods of a PageHandle. */
export const isPageHandle = (value: Unvalidated): value is PageHandle =>
  typeof value === 'object' &&
  value !== null &&
  'measure' in value &&
  typeof value.measure === 'function' &&
  'handleKey' in value &&
  typeof value.handleKey === 'function' &&
  'focusPages' in value &&
  typeof value.focusPages === 'function'

/** What one document tab holds. */
export type DocumentTabState = ReturnType<typeof useDocumentTab>

export function useDocumentTab(readItem: DocumentReaderState) {
  /** The page of this document, for as long as its tab is drawn. */
  const page = shallowRef<PageHandle | null>(null)

  const setPageHandle = (handle: PageHandle | null) => {
    page.value = handle
  }

  const measure = () => page.value?.measure()
  const handleKeyPress = (event: KeyboardEvent) => page.value?.handleKey(event) ?? false
  const focusTab = () => page.value?.focusPages()

  return {
    document: readItem,
    setPageHandle,
    measure,
    handleKeyPress,
    focusTab,
  }
}
