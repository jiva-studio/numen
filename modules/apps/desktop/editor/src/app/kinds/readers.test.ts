/** What the document reader kinds are handed. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createReaderKinds, type ReaderKindsDeps } from './readers'

const h = vi.hoisted(() => ({
  documentState: null as unknown as (path: string) => unknown,
}))

vi.mock('@/pages/document-viewer', () => ({
  documents: { name: 'documents' },
  createDocumentKind: (_handle: unknown, createState: (path: string) => unknown) => {
    h.documentState = createState
    return {}
  },
  useDocumentReader: (documents: unknown, path: string) => ({ documents, path }),
  useDocumentTab: (reader: unknown) => ({ tab: 'document', reader }),
}))

const setup = () => {
  createReaderKinds({
    tabOpeners: {},
    held: { handle: {} },
  } as unknown as ReaderKindsDeps)
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the document kind', () => {
  it('opens a tab over a reader of the documents at the path', () => {
    setup()

    expect(h.documentState('Doc.md')).toStrictEqual({
      tab: 'document',
      reader: { documents: { name: 'documents' }, path: 'Doc.md' },
    })
  })
})

