/**
 * Types and interfaces for book reading and navigation.
 */
import type { Span } from '@numen/plugin-sdk'

export type { Span }

/** One document of a book spine and its text range. */
export interface SpineDocument {
  readonly path: string
  readonly span: Span
}

/** One section or chapter named by a book. */
export interface BookPart {
  readonly title: string
  readonly offset: number
  readonly level: number
}

/** One page label of a printed edition. */
export interface PrintedPage {
  readonly label: string
  readonly offset: number
}

/** Book manifest and metadata. */
export interface Book {
  readonly title: string
  readonly span: Span
  readonly documents: readonly SpineDocument[]
  readonly parts: readonly BookPart[]
  readonly printedPages: readonly PrintedPage[]
  readonly pages: number
  readonly pageBytes: number
  readonly stat: string
}

/** Service port for book operations. */
export interface Books {
  /** Retrieves book metadata and spine. */
  getBook(path: string): Promise<Book>
  /** Reads document markup for a spine item. */
  readMarkup(path: string, document: string, stat: string): Promise<string>
  /** Returns the URL for an archived resource entry. */
  getEntryUrl(path: string, name: string, stat: string): string
  /** The EPUB CFI of the character at an offset of the text, and null for a book with no text. */
  getCfi(path: string, offset: number): Promise<string | null>
  /** The offset of the text a CFI names, and null where it names no place in the book. */
  findOffset(path: string, cfi: string): Promise<number | null>
}

/** Localization words for book navigation. */
export interface BookWords {
  readonly page: string
}

/** Port for interacting with the rendered book element. */
export interface BookHandle {
  measure(): void
  handleKey(event: KeyboardEvent): boolean
}
