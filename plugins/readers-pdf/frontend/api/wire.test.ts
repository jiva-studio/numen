/**
 * The pages of a document and where the text relies on them.
 *
 * What is proved here is the crossing: a layout comes back as the sizes and the
 * one string the window carries a file as, and a run comes back as its places,
 * a page each.
 */
import { describe, expect, it, vi } from 'vitest'
import { client, documents } from './wire'

const request = {
  getDocument: vi.fn(),
  getHighlights: vi.fn(),
}

client.documents = request as unknown as typeof client.documents

describe('the layout of a document', () => {
  it('arrives back as the size of every page and the file it was read from', async () => {
    request.getDocument.mockResolvedValue({
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
    expect(request.getDocument).toHaveBeenCalledWith({ path: 'Book.pdf' })
  })

  it('names no file where the answer carries none', async () => {
    request.getDocument.mockResolvedValue({ pages: [] })

    expect(await documents.getDocumentLayout('Book.pdf')).toEqual({ pages: [], fingerprint: '' })
  })
})

describe('the address a page is drawn from', () => {
  it('carries the width requested for and which bytes it is about', () => {
    expect(documents.getPageUrl('Books/Book.pdf', 3, 1200, '12 34 Books/Book.pdf')).toBe(
      '/plugin/md.numen.readers.pdf/pages/3?path=Books%2FBook.pdf&wide=1200&size=12&mtime=34',
    )
  })

  it('is about no bytes in particular where the caller read no file', () => {
    expect(documents.getPageUrl('Book.pdf', 1, 800)).toBe(
      '/plugin/md.numen.readers.pdf/pages/1?path=Book.pdf&wide=800&size=0&mtime=0',
    )
  })
})

describe('where a run of text is placed', () => {
  it('places a run on the pages and in the rectangles the answer lists', async () => {
    request.getHighlights.mockResolvedValue({
      runs: [
        {
          places: [
            {
              page: 2,
              rects: [
                { minX: 0.1, minY: 0.2, maxX: 0.3, maxY: 0.4 },
                { minX: 0.5, minY: 0.6, maxX: 0.7, maxY: 0.8 },
              ],
            },
            { page: 3, rects: [{ minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 }] },
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
        { page: 3, rects: [{ minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 }] },
      ],
    ])
    expect(request.getHighlights).toHaveBeenCalledWith({
      path: 'Book.pdf',
      spans: [{ from: 0, to: 9 }],
    })
  })

  it('places a run nowhere when the answer lists no places', async () => {
    request.getHighlights.mockResolvedValue({ runs: [{ places: [] }] })

    expect(await documents.getHighlights('Book.pdf', [{ from: 0, to: 9 }])).toEqual([[]])
  })

  it('answers one span at a time, and nothing for a span the answer skipped', async () => {
    request.getHighlights.mockResolvedValue({
      runs: [{ places: [{ page: 1, rects: [{ minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 }] }] }],
    })

    expect(
      await documents.getHighlights('Book.pdf', [
        { from: 0, to: 9 },
        { from: 10, to: 20 },
      ]),
    ).toEqual([[{ page: 1, rects: [{ minX: 0.1, minY: 0.1, maxX: 0.2, maxY: 0.2 }] }], []])
  })
})
