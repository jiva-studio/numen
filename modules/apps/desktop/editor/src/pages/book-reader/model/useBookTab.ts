/**
 * Tab state for book tabs.
 */
import { shallowRef } from 'vue'
import type { BookReaderState } from './useBookReader'
import type { BookHandle } from '../types'

export type { BookHandle }

/** What one book tab holds. */
export type BookTabState = ReturnType<typeof useBookTab>

export function useBookTab(read: BookReaderState) {
  const reader = shallowRef<BookHandle | null>(null)
  const tab = shallowRef<HTMLElement | null>(null)

  const setBookHandle = (handle: BookHandle | null) => {
    reader.value = handle
  }

  const setTabElement = (element: HTMLElement | null) => {
    tab.value = element
  }

  const measure = () => reader.value?.measure()
  const focusTab = () => tab.value?.focus()
  const handleKeyPress = (event: KeyboardEvent) => reader.value?.handleKey(event) ?? false

  return {
    ...read,
    setBookHandle,
    setTabElement,
    measure,
    focusTab,
    handleKeyPress,
  }
}
