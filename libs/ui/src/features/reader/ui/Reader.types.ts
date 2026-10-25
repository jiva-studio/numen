import type { Paint } from '@/shared/lib/time/clock'
import type { Page, ReaderWords, Rect } from '../lib/strip'

export interface ReaderProps {
  /** How big each page is, in its own units. */
  pages?: readonly Page[]
  /** Which page is in front, counted from the first. */
  at?: number
  /** Where one page is drawn, as an address to point a picture at. */
  picture?: (page: number) => string
  /** What is highlighted on one page, in fractions of it. */
  highlights?: (page: number) => readonly Rect[]
  /** The other places named on one page, apart from the one opened at. */
  otherHighlights?: (page: number) => readonly Rect[]
  /** The words it is read with. */
  words?: ReaderWords
  /** Text displayed when a page fails to load. */
  undrawn?: string
  /** When the row is put on the screen again. The next frame by default. */
  paint?: Paint
}
