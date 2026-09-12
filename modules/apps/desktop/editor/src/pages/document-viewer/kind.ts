/**
 * Window registration for document tabs.
 */
import type { Span } from '@/shared/span'
import type { FileOpeners } from '@/entities/tab'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { DOCUMENT } from '@/entities/tab'
import DocumentTab from './ui/DocumentTab.vue'
import { fileOf } from '@/shared/paths'
import type { DocumentTabState } from './model/useDocumentTab'

/**
 * The document tabs of a window. A document is its own tab, so the same one
 * opened again is the tab it is already read in.
 */
export function documentKind(handle: WindowHandle, opens: (path: string) => DocumentTabState, tabOpeners: FileOpeners) {
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

  const openDocument = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(DOCUMENT, path)
    void handle.holds<DocumentTabState>(DOCUMENT, id)?.focusSpans(...spans)
  }
  tabOpeners.registerReader({ kind: 'book' }, (path, spans) => void openDocument(path, spans))

  return { kind }
}
