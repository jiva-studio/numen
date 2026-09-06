/**
 * What one document tab holds: the document being read, and the page on screen.
 *
 * A page is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * measures again then.
 */
import type { OpenDocumentState } from './open'
import type { Stretch } from '../core'
import type { FileOpeners } from '../tabs/openers'
import type { Host, Kind } from '../tabs/windowing'
import { DOCUMENT } from '../tabs/workspace'
import DocumentTab from './DocumentTab.vue'

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
export function documentKind(host: Host, opens: (path: string) => DocumentTabState, puts: FileOpeners) {
  const kind: Kind<DocumentTabState> = {
    kind: DOCUMENT,
    opens,
    called: (held) => held.path.split('/').pop() ?? held.path,
    draws: DocumentTab,
    identity: (path) => path,
    shown: (held) => held.measure(),
    shuts: (held) => {
      held.close()
      return true
    },
    at: (held) => ({ file: held.path, source: 'book' }),
    attends: (held) => ({
      path: held.path,
      document: { page: held.at.value + 1, pages: held.pages.value },
    }),
  }

  // The reader of documents. What stands at the stretches asked for is
  // highlighted, and the tab turns to the first page of them; the rest are
  // highlighted where they fall, each of them somewhere else to look.
  const reads = async (path: string, stretches: readonly Stretch[]) => {
    const id = await host.opens(DOCUMENT, path)
    void host.holds<DocumentTabState>(DOCUMENT, id)?.reach(...stretches)
  }
  puts.reads((path, stretches) => void reads(path, stretches))

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
