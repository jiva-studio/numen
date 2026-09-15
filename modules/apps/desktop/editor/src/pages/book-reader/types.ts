/**
 * Types and interfaces for book reading and navigation.
 */
import type { Span } from '@/shared/span'

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
  readonly fingerprint: string
}

/** Service port for book operations. */
export interface Books {
  /** Retrieves book metadata and spine. */
  getBook(path: string): Promise<Book>
  /** Reads document markup for a spine item. */
  readMarkup(path: string, document: string, fingerprint: string): Promise<string>
  /** Returns the URL for an archived resource entry. */
  getEntryUrl(path: string, name: string, fingerprint: string): string
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
