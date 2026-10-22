import type { Span } from '@numen/plugin-sdk'

export interface BookContentProps {
  markup: string
  path: string
  documentSpan: Span
  bookSpan: Span
  offset: number
  highlights: readonly Span[]
  otherHighlights: readonly Span[]
  chapter: string
  isListingOpen: boolean
}
