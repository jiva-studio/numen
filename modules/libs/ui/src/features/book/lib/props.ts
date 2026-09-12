/**
 * What the reader of a book that reflows is told, before it draws anything.
 */
import type { BookWords, Span } from './spread'

/** What one document of a book reaches the reader as. */
export interface BookProps {
  /**
   * One document of the book, as it is drawn. Every run of text in it carries
   * `data-offset`, the byte offset at which that run begins in the book's text.
   * It reaches this component already measured against what may be drawn.
   */
  markup?: string
  /** The document being drawn, as the book's archive names it. */
  path?: string
  /** Where this document stands in the book, in bytes of the book's text. */
  span?: Span
  /** Where the book itself runs between, in bytes. */
  book?: Span
  /** The offset in front, in bytes of the book's text. */
  at?: number
  /** The runs highlighted where they stand, in bytes of the book's text. */
  highlights?: readonly Span[]
  /** The other runs asked about, each of them somewhere else to look. */
  elsewhere?: readonly Span[]
  /** How large the text is set, as a multiple of the size prose is read at. */
  textSize?: number
  /** What the book calls the place in front, drawn over the text it names. */
  chapter?: string
  /** The words it is read with. */
  words?: BookWords
}

/** What the reader works with, once what it was told stands at its defaults. */
export interface SettledBookProps {
  readonly markup: string
  readonly path: string
  readonly span: Span
  readonly book: Span
  readonly at: number
  readonly highlights: readonly Span[]
  readonly elsewhere: readonly Span[]
  readonly textSize: number
  readonly chapter: string
  readonly words: BookWords
}
