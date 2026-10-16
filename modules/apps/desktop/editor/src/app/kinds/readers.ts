/** The document reader tab kinds of a window. */
import {
  createDocumentKind,
  documents,
  useDocumentReader,
  useDocumentTab,
} from '@/pages/document-viewer'
import type { WindowKindsDeps } from './deps'

export type ReaderKindsDeps = Pick<WindowKindsDeps, 'tabOpeners' | 'held'>

export function createReaderKinds({ tabOpeners, held }: ReaderKindsDeps) {
  const read = createDocumentKind(
    held.handle,
    (path) => useDocumentTab(useDocumentReader(documents, path)),
    tabOpeners,
  )

  return { read }
}
