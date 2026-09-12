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
  })

  it('finds spine document containing offset', () => {
    const docs: SpineDocument[] = [
      { path: 'ch1.xhtml', span: { begins: 0, ends: 1000 } },
      { path: 'ch2.xhtml', span: { begins: 1000, ends: 2500 } },
      { path: 'ch3.xhtml', span: { begins: 2500, ends: 4000 } },
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
      span: { begins: 0, ends: 1000 },
      documents: [],
      parts: [{ title: 'Chapter 1', offset: 0, level: 0 }],
      printed: [],
      pages: 10,
      pageBytes: 100,
      fingerprint: 'f1',
    }

    expect(getContents(bookWithParts, words)).toEqual([
      { title: 'Chapter 1', at: 0, level: 0 },
    ])

    const bookWithPrinted: Book = {
      title: 'T',
      span: { begins: 0, ends: 1000 },
      documents: [],
      parts: [],
      printed: [{ label: 'iv', offset: 50 }],
      pages: 10,
      pageBytes: 100,
      fingerprint: 'f1',
    }

    expect(getContents(bookWithPrinted, words)).toEqual([
      { title: 'Page iv', at: 50, level: 0 },
    ])
  })
})
