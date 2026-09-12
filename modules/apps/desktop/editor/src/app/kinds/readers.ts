/** The document and book reader tab kinds of a window. */
import {
  documentKind,
  documents,
  useDocumentReader,
  useDocumentTab,
} from '@/pages/document-viewer'
import { bookKind, books, useBookReader, useBookTab, WORDS as bookWords } from '@/pages/book-reader'
import type { WindowKindsDeps } from './deps'

export type ReaderKindsDeps = Pick<WindowKindsDeps, 'log' | 'puts' | 'held'>

export function createReaderKinds({ log, puts, held }: ReaderKindsDeps) {
  const read = documentKind(
    held.handle,
    (path) => useDocumentTab(useDocumentReader(documents, path)),
    puts,
  )

  const turned = bookKind(
    held.handle,
    (path) => useBookTab(useBookReader(books, path, bookWords, log.under('book'))),
    puts,
  )

  return { read, turned }
}
