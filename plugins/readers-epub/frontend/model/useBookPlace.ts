/**
 * The place of a book tab as an EPUB CFI, the form the plugin's backend uses: found, saved and
 * reported to the window.
 */
import { watch, type Ref } from 'vue'
import type { Selector, TabPlace } from '@numen/plugin-sdk'
import { createCfiSelector, getSelectorCfi } from '../lib/place'

/** What the place is read from and moved through. */
export interface PlacedReader {
  readonly offset: Ref<number>
  readonly isLoading: Ref<boolean>
  /** The table-of-contents title at the offset, empty when the book has none. */
  readonly chapter: Ref<string>
  getCfi(offset: number): Promise<string | null>
  goToCfi(cfi: string): Promise<boolean>
}

/**
 * The tab as the window provides it: the selector it opens at and where it
 * saves its position and reports it to the window.
 */
export interface PlacedTab {
  readonly selector?: Selector | null
  savePlace?(selector: Selector): void
  reportPlace?(place: TabPlace): void
}

/** Returns the selector saved for a CFI: the CFI and the chapter title, omitted when empty. */
const createPlace = (cfi: string, chapter: string): Selector =>
  chapter === '' ? createCfiSelector(cfi) : { ...createCfiSelector(cfi), chapter }

export function useBookPlace(reader: PlacedReader, getTab: () => PlacedTab | undefined) {
  const reportPlace = (cfi: string, chapter: string) =>
    getTab()?.reportPlace?.(chapter === '' ? { cfi } : { cfi, chapter })

  // The window sets a selector for a search hit and for the place saved before a restart. The place
  // the tab moved to is reported at once, and it is not saved again.
  watch(
    () => [getTab()?.selector, reader.isLoading.value] as const,
    async ([selector, isLoading]) => {
      const cfi = getSelectorCfi(selector)
      if (isLoading || cfi === null) return
      if (await reader.goToCfi(cfi).catch(() => false)) reportPlace(cfi, reader.chapter.value)
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
      if (request !== turn || cfi === null) return
      const chapter = reader.chapter.value
      getTab()?.savePlace?.(createPlace(cfi, chapter))
      reportPlace(cfi, chapter)
    },
  )
}
