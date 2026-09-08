/**
 * What one document tab holds: the document being read, and the page on screen.
 *
 * A page is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * measures again then.
 */
import type { OpenDocumentState } from './open'
import type { Span } from '../../shared/core'
import type { FileOpeners } from '../../shared/tabs/openers'
import type { Kind, WindowHandle } from '../../shared/tabs/windowing'
import { DOCUMENT } from '../../shared/tabs/workspace'
import DocumentTab from './DocumentTab.vue'
import { fileOf } from '../../shared/paths'

/** What the window asks of a page once it is drawn. */
export interface PageHandle {
  measure(): void
}

/** What one document tab holds. */
export type DocumentTabState = ReturnType<typeof documenting>

/**
 * The document tabs of a window. A document is its own tab, so the same one
 * opened again is the tab it is already read in.
 */
export function documentKind(handle: WindowHandle, opens: (path: string) => DocumentTabState, puts: FileOpeners) {
  const kind: Kind<DocumentTabState> = {
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
    at: (state) => ({ file: state.path, source: 'book' }),
    attends: (state) => ({
      path: state.path,
      document: { page: state.at.value + 1, pageCount: state.pages.value.length },
    }),
  }

  // The reader of documents. What stands at the spans asked for is
  // highlighted, and the tab turns to the first page of them; the rest are
  // highlighted where they fall, each of them somewhere else to look.
  const reads = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(DOCUMENT, path)
    void handle.holds<DocumentTabState>(DOCUMENT, id)?.reach(...spans)
  }
  puts.reads((path, spans) => void reads(path, spans))

  return { kind }
}

export function documenting(read: OpenDocumentState) {
  /** The page of this document, for as long as its tab is drawn. */
  let page: PageHandle | null = null

  const drew = (drawn: unknown) => {
    page = (drawn as PageHandle | null) ?? null
  }

  const measure = () => page?.measure()

  return { ...read, drew, measure }
}
