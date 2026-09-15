/** The document and book reader tab kinds of a window. */
import { createDocumentKind, documents, useDocumentReader, useDocumentTab } from '@/pages/document-viewer'
import { createBookKind, books, useBookReader, useBookTab, WORDS as bookWords } from '@/pages/book-reader'
import type { WindowKindsDeps } from './deps'

export type ReaderKindsDeps = Pick<WindowKindsDeps, 'log' | 'tabOpeners' | 'held'>

export function createReaderKinds({ log, tabOpeners, held }: ReaderKindsDeps) {
  const read = createDocumentKind(
    held.handle,
    (path) => useDocumentTab(useDocumentReader(documents, path)),
    tabOpeners,
  )

  const turned = createBookKind(
    held.handle,
    (path) => useBookTab(useBookReader(books, path, bookWords, log.getWriter('book'))),
    tabOpeners,
  )

  return { read, turned }
}
