import type { TextRange } from '@numen/ui'
import type { BookWords } from './words'

/** The props of the reader for one document of a book. */
export interface BookProps {
  markup?: string
  path?: string
  span?: TextRange
  book?: TextRange
  at?: number
  target?: { readonly offset: number } | null
  highlights?: readonly TextRange[]
  otherHighlights?: readonly TextRange[]
  textSize?: number
  chapter?: string
  words?: BookWords
}

export interface SettledBookProps {
  readonly markup: string
  readonly path: string
  readonly span: TextRange
  readonly book: TextRange
  readonly at: number
  readonly target: { readonly offset: number } | null
  readonly highlights: readonly TextRange[]
  readonly otherHighlights: readonly TextRange[]
  readonly textSize: number
  readonly chapter: string
  readonly words: BookWords
}
