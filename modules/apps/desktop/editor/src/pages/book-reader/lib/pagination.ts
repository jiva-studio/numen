/**
 * Pagination, table of contents, and document lookup calculations.
 */
import type { ContentsEntry } from '@numen/ui'
import type { Book, BookWords, SpineDocument } from '../types'

export function getPageNumber(pageBytes: number, pages: number, offset: number): number {
  if (pages <= 0 || pageBytes <= 0) return 0
  return Math.min(Math.max(Math.floor(offset / pageBytes) + 1, 1), pages)
}

export function getDocumentAtOffset(
  documents: readonly SpineDocument[],
  offset: number,
): SpineDocument | undefined {
  let found: SpineDocument | undefined
  for (const one of documents) {
    if (one.span.begins > offset) break
    found = one
  }
  return found
}

function getDocumentTitle(document: SpineDocument): string {
  const last = document.path.split('/').pop() ?? document.path
  const dot = last.lastIndexOf('.')
  return dot > 0 ? last.slice(0, dot) : last
}

export function getContents(book: Book, words: BookWords): readonly ContentsEntry[] {
  if (book.parts.length !== 0) {
    return book.parts.map((one) => ({
      title: one.title,
      at: one.offset,
      level: one.level,
    }))
  }
  if (book.printed.length !== 0) {
    return book.printed.map((one) => ({
      title: `${words.page} ${one.label}`,
      at: one.offset,
      level: 0,
    }))
  }
  return book.documents.map((one) => ({ title: getDocumentTitle(one), at: one.span.begins, level: 0 }))
}
