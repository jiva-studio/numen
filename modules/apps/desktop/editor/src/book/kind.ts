/**
 * What one book tab holds: the book being read, and the document the offset in
 * front falls in.
 *
 * A book is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * lays the columns out again then.
 */
import type { OpenBookState } from './open'
import type { Stretch } from '../core'
import type { FileOpeners } from '../tabs/openers'
import type { Kind, WindowHandle } from '../tabs/windowing'
import { BOOK } from '../tabs/workspace'
import BookTab from './BookTab.vue'

/** What the window asks of a book once it is drawn. */
export interface BookHandle {
  measure(): void
}

/** What one book tab holds. */
export type BookTabState = ReturnType<typeof booking>

/**
 * The book tabs of a window. A book is its own tab, so the same one opened
 * again is the tab it is already read in.
 */
export function bookKind(
  handle: WindowHandle,
  opens: (path: string) => BookTabState,
  puts: FileOpeners,
) {
  const kind: Kind<BookTabState> = {
    kind: BOOK,
    opens,
    called: (state) => state.title.value || (state.path.split('/').pop() ?? state.path),
    draws: BookTab,
    identity: (path) => path,
    shown: (state) => state.measure(),
    shuts: (state) => {
      state.close()
      return true
    },
    at: (state) => ({ file: state.path, source: 'book' }),
    // Where a person is in a book that reflows is an offset into its text, and
    // the page beside it is the page that offset falls on.
    attends: (state) => ({
      path: state.path,
      book: {
        offset: state.at.value,
        length: state.span.value.ends,
        page: state.page.value,
        pages: state.pages.value,
      },
    }),
  }

  // The reader of books that reflow. The stretch asked for is marked where it
  // stands and the tab is turned to it; the rest are marked more faintly
  // wherever they fall, each of them somewhere else to look.
  const turns = async (path: string, stretches: readonly Stretch[]) => {
    const id = await handle.opens(BOOK, path)
    void handle.holds<BookTabState>(BOOK, id)?.reach(...stretches)
  }
  puts.turns((path, stretches) => void turns(path, stretches))

  return { kind }
}

export function booking(read: OpenBookState) {
  /** The book as it is drawn, for as long as its tab is drawn. */
  let reader: BookHandle | null = null

  const drew = (held: unknown) => {
    reader = (held as BookHandle | null) ?? null
  }

  const measure = () => reader?.measure()

  return { ...read, drew, measure }
}
