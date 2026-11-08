import type { Paint } from '@numen/ui'
import type { Fit, Page, ReaderWords, Rect } from '../lib/strip'

export interface ReaderProps {
  pages?: readonly Page[]
  at?: number
  picture?: (page: number) => string
  highlights?: (page: number) => readonly Rect[]
  otherHighlights?: (page: number) => readonly Rect[]
  words?: ReaderWords
  undrawn?: string
  paint?: Paint
  fit?: Fit
}
