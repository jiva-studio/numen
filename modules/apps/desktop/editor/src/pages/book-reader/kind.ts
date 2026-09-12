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
  open: (path: string) => BookTabState,
  tabOpeners: FileOpeners,
) {
  const kind: TabKind<BookTabState, typeof BOOK> = {
    kind: BOOK,
    open,
    getTitle: (state) => state.title.value || (state.path.split('/').pop() ?? state.path),
    pane: BookTab,
    identity: (path) => path,
    onShow: (state) => {
      state.measure()
      state.focusTab()
    },
    onKeyPress: (state, event) => state.handleKeyPress(event),
    onClose: (state) => {
      state.close()
      return true
    },
    over: (state) => ({ file: state.path, source: 'book' }),
    getOpenTab: (state) => ({
      path: state.path,
      book: {
        offset: state.offset.value,
        page: state.page.value,
        pageCount: state.pages.value,
      },
    }),
  }

  const openBook = async (path: string, spans: readonly Span[]) => {
    const id = await handle.openTab(BOOK, path)
    void handle.getTabState<BookTabState>(BOOK, id)?.focusSpans(...spans)
  }
  tabOpeners.registerReader({ kind: 'book', format: 'epub' }, (path, spans) => void openBook(path, spans))

  return { kind }
}
