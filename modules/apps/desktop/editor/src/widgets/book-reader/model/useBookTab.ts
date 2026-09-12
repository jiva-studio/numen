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
