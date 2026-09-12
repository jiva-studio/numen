/**
 * Window registration and tab state for book tabs.
 */
import { shallowRef } from 'vue'
import type { BookReaderState } from './useBookReader'
import type { BookHandle } from '../types'
import type { Span } from '@/shared/span'
import type { FileOpeners } from '@/entities/tab/openers'
import type { TabKind, WindowHandle } from '@/entities/tab/windowTabs'
import { BOOK } from '@/entities/tab/workspace'
import BookTab from '../components/BookTab.vue'

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

export function useBookTab(read: BookReaderState) {
  const reader = shallowRef<BookHandle | null>(null)
  const tab = shallowRef<HTMLElement | null>(null)

  const setBookHandle = (held: BookHandle | null) => {
    reader.value = held
  }

  const setTabElement = (held: HTMLElement | null) => {
    tab.value = held
  }

  const measure = () => reader.value?.measure()
  const focusTab = () => tab.value?.focus()
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
