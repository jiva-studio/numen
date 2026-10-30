import type { TextRange } from '@numen/plugin-sdk'

export interface BookContentProps {
  markup: string
  path: string
  documentSpan: TextRange
  bookSpan: TextRange
  offset: number
  /** The latest jump to an offset. The book shows its spread when the offset is in the drawn document. */
  target: { readonly offset: number } | null
  highlights: readonly TextRange[]
  otherHighlights: readonly TextRange[]
  chapter: string
  isListingOpen: boolean
}
