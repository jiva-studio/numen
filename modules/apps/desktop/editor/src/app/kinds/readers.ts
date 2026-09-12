/** The document and book reader tab kinds of a window. */
import {
  documentKind,
  documents,
  useDocumentReader,
  useDocumentTab,
} from '@/pages/document-viewer'
import { bookKind, books, useBookReader, useBookTab, WORDS as bookWords } from '@/pages/book-reader'
import type { WindowKindsDeps } from './deps'

export type ReaderKindsDeps = Pick<WindowKindsDeps, 'log' | 'tabOpeners' | 'held'>

export function createReaderKinds({ log, tabOpeners, held }: ReaderKindsDeps) {
  const read = documentKind(
    held.handle,
    (path) => useDocumentTab(useDocumentReader(documents, path)),
    tabOpeners,
  )

  const turned = bookKind(
    held.handle,
    (path) => useBookTab(useBookReader(books, path, bookWords, log.under('book'))),
    tabOpeners,
  )

  return { read, turned }
}
