/** What the document and book reader kinds are handed. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createReaderKinds, type ReaderKindsDeps } from './readers'

const h = vi.hoisted(() => ({
  documentState: null as unknown as (path: string) => unknown,
  bookState: null as unknown as (path: string) => unknown,
  useBookReader: vi.fn((..._args: unknown[]) => ({ reader: 'book reader' })),
  useBookTab: vi.fn((reader: unknown) => ({ tab: 'book', reader })),
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
vi.mock('@/pages/book-reader', () => ({
  books: { name: 'books' },
  WORDS: { name: 'book words' },
  createBookKind: (_handle: unknown, createState: (path: string) => unknown) => {
    h.bookState = createState
    return {}
  },
  useBookReader: h.useBookReader,
  useBookTab: h.useBookTab,
}))

const setup = () => {
  const writer = vi.fn()
  const getWriter = vi.fn(() => writer)
  createReaderKinds({
    log: { getWriter },
    tabOpeners: {},
    held: { handle: {} },
  } as unknown as ReaderKindsDeps)
  return { writer, getWriter }
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

describe('the book kind', () => {
  it('opens a tab over a reader of the books at the path, writing to the book log', () => {
    const one = setup()

    const state = h.bookState('Book.epub')

    expect(one.getWriter).toHaveBeenCalledWith('book')
    expect(h.useBookReader).toHaveBeenCalledWith(
      { name: 'books' },
      'Book.epub',
      { name: 'book words' },
      one.writer,
    )
    expect(state).toStrictEqual({ tab: 'book', reader: { reader: 'book reader' } })
  })
})
