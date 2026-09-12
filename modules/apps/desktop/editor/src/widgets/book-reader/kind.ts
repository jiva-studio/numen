/**
 * Window registration for book tabs.
 */
import type { Span } from '@/shared/span'
import type { FileOpeners } from '@/entities/tab'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { BOOK } from '@/entities/tab'
import BookTab from './ui/BookTab.vue'
import type { BookTabState } from './model/useBookTab'

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
    presses: (state, event) => state.handleKeyPress(event),
    shuts: (state) => {
      state.close()
      return true
    },
    over: (state) => ({ file: state.path, source: 'book' }),
    attends: (state) => ({
      path: state.path,
      book: {
        offset: state.offset.value,
        page: state.page.value,
        pageCount: state.pages.value,
      },
    }),
    getAttention: (state) => ({
      path: state.path,
      book: {
        offset: state.offset.value,
        page: state.page.value,
        pageCount: state.pages.value,
      },
    }),
  }

  const turns = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(BOOK, path)
    void handle.holds<BookTabState>(BOOK, id)?.focusSpans(...spans)
  }
  puts.reads({ kind: 'book', format: 'epub' }, (path, spans) => void turns(path, spans))

  return { kind }
}
