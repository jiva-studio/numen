/**
 * What one document tab holds: the document being read, and the page on screen.
 *
 * A page is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * measures again then.
 */
import type { Reading } from '../reading'

/** What the window asks of a page once it is drawn. */
export interface Drawn {
  measure(): void
}

/** What one document tab holds. */
export type Held = ReturnType<typeof documenting>

export function documenting(read: Reading) {
  /** The page of this document, for as long as its tab is drawn. */
  let page: Drawn | null = null

  const drew = (drawn: unknown) => {
    page = (drawn as Drawn | null) ?? null
  }

  const measure = () => page?.measure()

  return { ...read, drew, measure }
}
