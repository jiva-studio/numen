/**
 * The place of a book tab, found and saved as the EPUB CFI the plugin's backend names it by.
 */
import { watch, type Ref } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import { createCfiSelector, getSelectorCfi } from '../lib/place'

/** What the place is read from and moved through. */
export interface PlacedReader {
  readonly offset: Ref<number>
  readonly isLoading: Ref<boolean>
  /** The table-of-contents title at the offset, empty where the book names none. */
  readonly chapter: Ref<string>
  getCfi(offset: number): Promise<string | null>
  goToCfi(cfi: string): Promise<boolean>
}

/**
 * The tab as the window provides it: the selector it opens at and where it
 * saves its position.
 */
export interface PlacedTab {
  readonly selector?: Selector | null
  savePlace?(selector: Selector): void
}

/** The place saved for a CFI: the CFI and the chapter title, the title left out where it is empty. */
const createPlace = (cfi: string, chapter: string): Selector =>
  chapter === '' ? createCfiSelector(cfi) : { ...createCfiSelector(cfi), chapter }

export function useBookPlace(reader: PlacedReader, getTab: () => PlacedTab | undefined) {
  // The window sets a selector for a search hit and for the place saved before a restart.
  watch(
    () => [getTab()?.selector, reader.isLoading.value] as const,
    ([selector, isLoading]) => {
      const cfi = getSelectorCfi(selector)
      if (!isLoading && cfi !== null) reader.goToCfi(cfi).catch(() => false)
    },
    { immediate: true },
  )

  // The answer for an offset the reader has already moved past is not saved.
  let turn = 0
  watch(
    () => reader.offset.value,
    async (offset) => {
      if (reader.isLoading.value) return
      turn += 1
      const request = turn
      const cfi = await reader.getCfi(offset).catch(() => null)
      if (request === turn && cfi !== null) getTab()?.savePlace?.(createPlace(cfi, reader.chapter.value))
    },
  )
}
