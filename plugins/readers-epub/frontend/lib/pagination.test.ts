import { describe, expect, it } from 'vitest'
import { getContents, getDocumentAtOffset, getPageNumber } from './pagination'
import type { Book, SpineDocument } from '../types'

describe('pagination and document lookup', () => {
  it('calculates page number based on byte offset', () => {
    expect(getPageNumber(1000, 10, 0)).toBe(1)
    expect(getPageNumber(1000, 10, 999)).toBe(1)
    expect(getPageNumber(1000, 10, 1000)).toBe(2)
    expect(getPageNumber(1000, 10, 50000)).toBe(10)
    expect(getPageNumber(0, 10, 500)).toBe(0)
    expect(getPageNumber(1000, 0, 500)).toBe(0)
    expect(getPageNumber(1000, -5, 500)).toBe(0)
  })

  it('finds spine document containing offset', () => {
    const docs: SpineDocument[] = [
      { path: 'ch1.xhtml', span: { start: 0, end: 1000 } },
      { path: 'ch2.xhtml', span: { start: 1000, end: 2500 } },
      { path: 'ch3.xhtml', span: { start: 2500, end: 4000 } },
    ]

    expect(getDocumentAtOffset(docs, 0)?.path).toBe('ch1.xhtml')
    expect(getDocumentAtOffset(docs, 500)?.path).toBe('ch1.xhtml')
    expect(getDocumentAtOffset(docs, 1000)?.path).toBe('ch2.xhtml')
    expect(getDocumentAtOffset(docs, 3000)?.path).toBe('ch3.xhtml')
  })

  it('builds table of contents entries from parts, printed pages, or spine docs', () => {
    const words = { page: 'Page' }
    const bookWithParts: Book = {
      title: 'T',
      span: { start: 0, end: 1000 },
      documents: [],
      parts: [{ title: 'Chapter 1', offset: 0, level: 0 }],
      printedPages: [],
      pages: 10,
      pageBytes: 100,
      stat: 'f1',
    }

    expect(getContents(bookWithParts, words)).toEqual([{ title: 'Chapter 1', at: 0, level: 0 }])

    const bookWithPrinted: Book = {
      title: 'T',
      span: { start: 0, end: 1000 },
      documents: [],
      parts: [],
      printedPages: [{ label: 'iv', offset: 50 }],
      pages: 10,
      pageBytes: 100,
      stat: 'f1',
    }

    expect(getContents(bookWithPrinted, words)).toEqual([{ title: 'Page iv', at: 50, level: 0 }])
  })
})

describe('the name of a document of the spine', () => {
  const book = (path: string): Book => ({
    title: 'T',
    span: { start: 0, end: 10 },
    documents: [{ path, span: { start: 0, end: 10 } }],
    parts: [],
    printedPages: [],
    pages: 1,
    pageBytes: 10,
    stat: 'f',
  })
  const getTitle = (path: string) => getContents(book(path), { page: 'Page' })[0]?.title

  it('is the file name without its extension', () => {
    expect(getTitle('text/part0001.xhtml')).toBe('part0001')
  })

  it('is the whole file name where there is no extension', () => {
    expect(getTitle('text/README')).toBe('README')
  })

  it('is the whole file name where the only dot begins it', () => {
    expect(getTitle('text/.hidden')).toBe('.hidden')
  })
})
