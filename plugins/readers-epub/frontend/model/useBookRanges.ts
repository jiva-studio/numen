/**
 * The ranges of a book tab: the runs of the book's text the window requests the tab to highlight.
 */
import { watch, type Ref } from 'vue'
import type { TextRange } from '../types'
import type { Caught } from '@numen/plugin-sdk'

/** The reader that highlights the ranges. */
export interface RangeReader {
  readonly isLoading: Ref<boolean>
  focusSpans(...spans: readonly TextRange[]): Promise<void>
  reportFailure(error: Caught): void
}

/** The tab as the window provides it: the ranges to highlight; the reader scrolls to the first. */
export interface RangeTab {
  readonly ranges?: readonly TextRange[] | null
}

export function useBookRanges(reader: RangeReader, getTab: () => RangeTab | undefined): void {
  // The window sets a new array on each request; each array is highlighted once, after the book loads.
  let highlightedRanges: readonly TextRange[] | null = null
  watch(
    () => [getTab()?.ranges, reader.isLoading.value] as const,
    ([ranges, isLoading]) => {
      if (isLoading || !ranges || ranges.length === 0 || ranges === highlightedRanges) return
      highlightedRanges = ranges
      reader.focusSpans(...ranges).catch(reader.reportFailure)
    },
    { immediate: true },
  )
}
