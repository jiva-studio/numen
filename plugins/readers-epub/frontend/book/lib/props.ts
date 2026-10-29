import type { TextRange } from '@numen/ui'
import type { BookWords } from './words'

/** The props of the reader for one document of a book. */
export interface BookProps {
  /**
   * One document of the book, as it is drawn. Every run of text in it carries
   * `data-offset`, the byte offset at which that run begins in the book's text.
   * It arrives at this component already measured against the drawable content.
   */
  markup?: string
  /** The document being drawn, as the book's archive names it. */
  path?: string
  span?: TextRange
  /** Where the book itself runs between, in bytes. */
  book?: TextRange
  /** The offset in front, in bytes of the book's text. */
  at?: number
  /** The runs highlighted where they stand, in bytes of the book's text. */
  highlights?: readonly TextRange[]
  /** The other runs named, apart from the one the reader was opened at. */
  otherHighlights?: readonly TextRange[]
  /** How large the text is set, as a multiple of the size prose is read at. */
  textSize?: number
  /** What the book calls the place in front, drawn over the text it names. */
  chapter?: string
  /** The words it is read with. */
  words?: BookWords
}

export interface SettledBookProps {
  readonly markup: string
  readonly path: string
  readonly span: TextRange
  readonly book: TextRange
  readonly at: number
  readonly highlights: readonly TextRange[]
  readonly otherHighlights: readonly TextRange[]
  readonly textSize: number
  readonly chapter: string
  readonly words: BookWords
}
