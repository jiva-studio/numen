/**
 * What one book tab holds: the book being read, and the document the offset in
 * front falls in.
 *
 * A book is laid out against the room it has, and a tab is drawn while it is
 * out of sight, where there is none. What is drawn says so when it appears, and
 * lays the columns out again then.
 */
import type { OpenBookState } from './open'
import type { Span } from '../shared/core'
import type { FileOpeners } from '../shared/tabs/openers'
import type { Kind, WindowHandle } from '../shared/tabs/windowTabs'
import { BOOK } from '../shared/tabs/workspace'
import BookTab from './BookTab.vue'

/** What the window asks of a book once it is drawn. */
export interface BookHandle {
  measure(): void
  /** A key the tab caught: true where it turned the page. */
  pressed(event: KeyboardEvent): boolean
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
    shown: (state) => {
      state.measure()
      state.takes()
    },
    // Several panes are drawn at once, so the book asked is the one in the pane
    // the person is in.
    presses: (state, event) => state.pressed(event),
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
        page: state.page.value,
        pageCount: state.pages.value,
      },
    }),
  }

  // The reader of books that reflow. The stretch asked for is marked where it
  // stands and the tab is turned to it; the rest are marked more faintly
  // wherever they fall, each of them somewhere else to look.
  const turns = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(BOOK, path)
    void handle.holds<BookTabState>(BOOK, id)?.reach(...spans)
  }
  puts.turns((path, spans) => void turns(path, spans))

  return { kind }
}

export function booking(read: OpenBookState) {
  /** The book as it is drawn, for as long as its tab is drawn. */
  let reader: BookHandle | null = null

  /** The tab it is drawn in, which is what holds the keyboard. */
  let tab: HTMLElement | null = null

  const drew = (held: unknown) => {
    reader = (held as BookHandle | null) ?? null
  }

  const stands = (held: HTMLElement | null) => {
    tab = held
  }

  const measure = () => reader?.measure()

  /**
   * The book takes the keyboard, the way a note opened takes it. What holds it
   * reads the arrows for itself: the tree walks its rows by them, and a book
   * opened out of the tree and left without it is a book the arrows never
   * reach.
   */
  const takes = () => tab?.focus()

  /** A key the tab caught, answered by the book it is drawn in. */
  const pressed = (event: KeyboardEvent) => reader?.pressed(event) ?? false

  return { ...read, drew, stands, measure, takes, pressed }
}
