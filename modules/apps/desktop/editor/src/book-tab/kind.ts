/**
 * Window registration and tab state for book tabs.
 */
import { shallowRef } from 'vue'
import type { BookReaderState } from './open'
import type { BookHandle } from './types'
import type { Span } from '../shared/core'
import type { FileOpeners } from '../shared/tabs/openers'
import type { TabKind, WindowHandle } from '../shared/tabs/windowTabs'
import { BOOK } from '../shared/tabs/workspace'
import BookTab from './BookTab.vue'

export type { BookHandle }


/** What one book tab holds. */
export type BookTabState = ReturnType<typeof useBookTab>

/**
 * The book tabs of a window. A book is its own tab, so the same one opened
 * again is the tab it is already read in.
 */
export function bookKind(
  handle: WindowHandle,
  opens: (path: string) => BookTabState,
  puts: FileOpeners,
) {
  const kind: TabKind<BookTabState, typeof BOOK> = {
    kind: BOOK,
    opens,
    called: (state) => state.title.value || (state.path.split('/').pop() ?? state.path),
    draws: BookTab,
    identity: (path) => path,
    shown: (state) => {
      state.measure()
      state.focusTab()
    },
    // Several panes are drawn at once, so the book asked is the one in the pane
    // the person is in.
    presses: (state, event) => state.handleKeyPress(event),
    shuts: (state) => {
      state.close()
      return true
    },
    over: (state) => ({ file: state.path, source: 'book' }),
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
    getAttention: (state) => ({
      path: state.path,
      book: {
        offset: state.at.value,
        page: state.page.value,
        pageCount: state.pages.value,
      },
    }),
  }

  // The reader of books that reflow. The span asked for is marked where it
  // stands and the tab is turned to it; the rest are marked more faintly
  // wherever they fall, each of them somewhere else to look.
  const turns = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(BOOK, path)
    void handle.holds<BookTabState>(BOOK, id)?.reach(...spans)
  }
  // A book that reflows is this kind's to read, and no other book's.
  puts.reads({ kind: 'book', format: 'epub' }, (path, spans) => void turns(path, spans))

  return { kind }
}

export function useBookTab(read: BookReaderState) {
  /** The book as it is drawn, for as long as its tab is drawn. */
  const reader = shallowRef<BookHandle | null>(null)

  /** The tab it is drawn in, which is what holds the keyboard. */
  const tab = shallowRef<HTMLElement | null>(null)

  /** The book the tab is drawing, and nothing while it draws none. */
  const setBookHandle = (held: BookHandle | null) => {
    reader.value = held
  }

  /** The tab as it is drawn, which is the element the keyboard is brought to. */
  const setTabElement = (held: HTMLElement | null) => {
    tab.value = held
  }

  const measure = () => reader.value?.measure()

  /**
   * The book takes the keyboard, the way a note opened takes it. What holds it
   * reads the arrows for itself: the tree walks its rows by them, and a book
   * opened out of the tree and left without it is a book the arrows never
   * reach.
   */
  const focusTab = () => tab.value?.focus()

  /** A key the tab caught, answered by the book it is drawn in. */
  const handleKeyPress = (event: KeyboardEvent) => reader.value?.pressed(event) ?? false

  return {
    ...read,
    setBookHandle,
    setTabElement,
    measure,
    focusTab,
    handleKeyPress,
  }
}

