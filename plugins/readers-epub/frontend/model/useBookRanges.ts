/**
 * The ranges of a book tab: the runs of the book's text the window asks the tab to light.
 */
import { watch, type Ref } from 'vue'
import type { TextRange } from '../types'

/** What the ranges are lit through. */
export interface RangedReader {
  readonly isLoading: Ref<boolean>
  focusSpans(...spans: readonly TextRange[]): Promise<void>
}

/** The tab as the window provides it: the ranges it is asked to light, the first to go to. */
export interface RangedTab {
  readonly ranges?: readonly TextRange[] | null
}

export function useBookRanges(reader: RangedReader, getTab: () => RangedTab | undefined): void {
  // The window sets a new array each time it asks for ranges; one array is lit once, after the book loads.
  let litRanges: readonly TextRange[] | null = null
  watch(
    () => [getTab()?.ranges, reader.isLoading.value] as const,
    ([ranges, isLoading]) => {
      if (isLoading || !ranges || ranges.length === 0 || ranges === litRanges) return
      litRanges = ranges
      reader.focusSpans(...ranges).catch(() => undefined)
    },
    { immediate: true },
  )
}
