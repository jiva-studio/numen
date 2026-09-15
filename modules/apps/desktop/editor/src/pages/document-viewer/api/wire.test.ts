/**
 * The pages of a document and where the text stands on them.
 *
 * What is proved here is the crossing: a layout comes back as the sizes and the
 * one string the window carries a file as, and the boxes of a run gather into
 * one entry for each page they fall on.
 */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({
  documents: asked,
  ocr: asked,
}))

const asked = {
  getDocument: vi.fn(),
  readOcr: vi.fn(),
}

const { documents } = await import('./wire')

describe('the layout of a document', () => {
  it('comes back as the size of every page and the file it was read from', async () => {
    asked.getDocument.mockResolvedValue({
      pages: [
        { width: 612, height: 792 },
        { width: 595, height: 842 },
      ],
      fingerprint: { path: 'Book.pdf', size: 12n, mtime: 34n },
    })

    expect(await documents.getDocumentLayout('Book.pdf')).toEqual({
      pages: [
        { width: 612, height: 792 },
        { width: 595, height: 842 },
      ],
      fingerprint: '12 34 Book.pdf',
    })
    expect(asked.getDocument).toHaveBeenCalledWith({ path: 'Book.pdf' })
  })

  it('names no file where the answer carries none', async () => {
    asked.getDocument.mockResolvedValue({ pages: [] })

    expect(await documents.getDocumentLayout('Book.pdf')).toEqual({ pages: [], fingerprint: '' })
  })
})

describe('the address a page is drawn from', () => {
  it('carries the width asked for and which bytes it is about', () => {
    expect(documents.getPageUrl('Books/Book.pdf', 3, 1200, '12 34 Books/Book.pdf')).toBe(
      '/assets/Books%2FBook.pdf/pages/3?wide=1200&size=12&mtime=34',
    )
  })

  it('is about no bytes in particular where the caller read no file', () => {
    expect(documents.getPageUrl('Book.pdf', 1, 800)).toBe(
      '/assets/Book.pdf/pages/1?wide=800&size=0&mtime=0',
    )
  })
})

describe('where a run of text stands', () => {
  it('gathers the boxes of one page into a single entry', async () => {
    asked.readOcr.mockResolvedValue({
      runs: [
        {
          boxes: [
            { page: 2, rect: { minX: 0.1, minY: 0.2, maxX: 0.3, maxY: 0.4 } },
            { page: 2, rect: { minX: 0.5, minY: 0.6, maxX: 0.7, maxY: 0.8 } },
          ],
        },
      ],
    })

    expect(await documents.getHighlights('Book.pdf', [{ from: 0, to: 9 }])).toEqual([
      [
        {
          page: 2,
          rects: [
            { minX: 0.1, minY: 0.2, maxX: 0.3, maxY: 0.4 },
            { minX: 0.5, minY: 0.6, maxX: 0.7, maxY: 0.8 },
          ],
        },
      ],
    ])
    expect(asked.readOcr).toHaveBeenCalledWith({ path: 'Book.pdf', spans: [{ from: 0, to: 9 }] })
  })

  it('opens a new entry where a box falls on another page', async () => {
    asked.readOcr.mockResolvedValue({
      runs: [
        {
          boxes: [
            { page: 2, rect: { minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 } },
            { page: 3, rect: { minX: 0.3, minY: 0.3, maxX: 0.4, maxY: 0.4 } },
            { page: 2, rect: { minX: 0.5, minY: 0.5, maxX: 0.6, maxY: 0.6 } },
          ],
        },
      ],
    })

    const [pages = []] = await documents.getHighlights('Book.pdf', [{ from: 0, to: 9 }])

    expect(pages.map((one) => one.page)).toEqual([2, 3, 2])
    expect(pages.map((one) => one.rects.length)).toEqual([1, 1, 1])
  })

  it('stands nowhere where the run carries no boxes', async () => {
    asked.readOcr.mockResolvedValue({ runs: [{ boxes: [] }] })

    expect(await documents.getHighlights('Book.pdf', [{ from: 0, to: 9 }])).toEqual([[]])
  })

  it('stands at the corner where a box carries no rectangle', async () => {
    asked.readOcr.mockResolvedValue({ runs: [{ boxes: [{ page: 1 }] }] })

    expect(await documents.getHighlights('Book.pdf', [{ from: 0, to: 9 }])).toEqual([
      [{ page: 1, rects: [{ minX: 0, minY: 0, maxX: 0, maxY: 0 }] }],
    ])
  })

  it('answers one span at a time, and nothing for a span the answer skipped', async () => {
    asked.readOcr.mockResolvedValue({
      runs: [{ boxes: [{ page: 1, rect: { minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 } }] }],
    })

    expect(
      await documents.getHighlights('Book.pdf', [
        { from: 0, to: 9 },
        { from: 10, to: 20 },
      ]),
    ).toEqual([[{ page: 1, rects: [{ minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 }] }], []])
  })
})
