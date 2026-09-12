/**
 * Window registration and tab state for document tabs.
 */
import { shallowRef } from 'vue'
import type { DocumentReaderState } from './useDocumentReader'
import type { PageHandle } from '../types'
import type { Span } from '@/shared/span'
import type { FileOpeners } from '@/entities/tab/openers'
import type { TabKind, WindowHandle } from '@/entities/tab/windowTabs'
import { DOCUMENT } from '@/entities/tab/workspace'
import DocumentTab from '../components/DocumentTab.vue'
import { fileOf } from '@/shared/paths'

export type { PageHandle }

/** What one document tab holds. */
export type DocumentTabState = ReturnType<typeof useDocumentTab>

/**
 * The document tabs of a window. A document is its own tab, so the same one
 * opened again is the tab it is already read in.
 */
export function documentKind(handle: WindowHandle, opens: (path: string) => DocumentTabState, puts: FileOpeners) {
  const kind: TabKind<DocumentTabState, typeof DOCUMENT> = {
    kind: DOCUMENT,
    opens,
    called: (state) => fileOf(state.path),
    draws: DocumentTab,
    identity: (path) => path,
    shown: (state) => state.measure(),
    shuts: (state) => {
      state.close()
      return true
    },
    over: (state) => ({ file: state.path, source: 'book' }),
    attends: (state) => ({
      path: state.path,
      document: { page: state.pageNumber.value + 1, pageCount: state.pages.value.length },
    }),
    getAttention: (state) => ({
      path: state.path,
      document: { page: state.pageNumber.value + 1, pageCount: state.pages.value.length },
    }),
  }

  const reads = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(DOCUMENT, path)
    void handle.holds<DocumentTabState>(DOCUMENT, id)?.focusSpans(...spans)
  }
  puts.reads({ kind: 'book' }, (path, spans) => void reads(path, spans))

  return { kind }
}

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
