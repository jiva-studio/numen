import type { TextRange } from '@numen/plugin-sdk'

export interface BookContentProps {
  markup: string
  path: string
  documentSpan: TextRange
  bookSpan: TextRange
  offset: number
  highlights: readonly TextRange[]
  otherHighlights: readonly TextRange[]
  chapter: string
  isListingOpen: boolean
}
