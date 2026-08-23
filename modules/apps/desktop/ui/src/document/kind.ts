/**
 * What one document tab holds: the document being read, and the page on screen.
 *
 * A page is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * measures again then.
 */
import type { Reading } from '../reading'
import type { Kind } from '../windowing'
import { DOCUMENT } from '../workspace'
import DocumentTab from './DocumentTab.vue'

/** What the window asks of a page once it is drawn. */
export interface Drawn {
  measure(): void
}

/** What one document tab holds. */
export type Held = ReturnType<typeof documenting>

/**
 * A document tab as the window keeps it. A document is its own tab, so the
 * same one opened again is the tab it is already read in.
 */
export const documentKind = (opens: (path: string) => Held): Kind<Held> => ({
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
})

export function documenting(read: Reading) {
  /** The page of this document, for as long as its tab is drawn. */
  let page: Drawn | null = null

  const drew = (drawn: unknown) => {
    page = (drawn as Drawn | null) ?? null
  }

  const measure = () => page?.measure()

  return { ...read, drew, measure }
}
