/**
 * The ranges of a book tab: the runs of the book's text the window asks the tab to light.
 */
import type { Ref } from 'vue'
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

export function useBookRanges(_reader: RangedReader, _getTab: () => RangedTab | undefined): void {
  // The stub lights no range.
}
