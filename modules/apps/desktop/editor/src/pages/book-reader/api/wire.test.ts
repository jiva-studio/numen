/**
 * The spine of a book and the markup of one document of it.
 *
 * What is proved here is the crossing: a document of the spine arrives as the
 * range of the text it covers, and the file the caller read goes back out as
 * the three parts the schema holds it in, or not at all.
 */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({
  books: asked,
}))

const asked = {
  getBook: vi.fn(),
  readBookMarkup: vi.fn(),
}

const { books } = await import('./wire')

/** An answer with nothing in it, for a test that cares about one field only. */
const nothing = {
  title: '',
  textBytes: 0,
  documents: [],
  parts: [],
  printedPages: [],
  pageCount: 0,
  pageBytes: 0,
}

describe('a book', () => {
  it('comes back as its spine, its parts and the file it was read from', async () => {
    asked.getBook.mockResolvedValue({
      title: 'Anabasis',
      textBytes: 4000,
      documents: [
        { path: 'one.xhtml', offset: 0, length: 1500 },
        { path: 'two.xhtml', offset: 1500, length: 2500 },
      ],
      parts: [{ title: 'Book I', offset: 0, level: 1 }],
      printedPages: [{ label: 'iv', offset: 120 }],
      pageCount: 8,
      pageBytes: 500,
      fingerprint: { path: 'Anabasis.epub', size: 12n, mtime: 34n },
    })

    expect(await books.getBook('Anabasis.epub')).toEqual({
      title: 'Anabasis',
      span: { from: 0, to: 4000 },
      documents: [
        { path: 'one.xhtml', span: { from: 0, to: 1500 } },
        { path: 'two.xhtml', span: { from: 1500, to: 4000 } },
      ],
      parts: [{ title: 'Book I', offset: 0, level: 1 }],
      printedPages: [{ label: 'iv', offset: 120 }],
      pages: 8,
      pageBytes: 500,
      fingerprint: '12 34 Anabasis.epub',
    })
    expect(asked.getBook).toHaveBeenCalledWith({ path: 'Anabasis.epub' })
  })

  it('names no file where the answer carries none', async () => {
    asked.getBook.mockResolvedValue(nothing)

    expect((await books.getBook('Anabasis.epub')).fingerprint).toBe('')
  })
})

describe('the markup of one document', () => {
  it('names the file the window read', async () => {
    asked.readBookMarkup.mockResolvedValue({ markup: '<p>Up country</p>' })

    expect(await books.readMarkup('Anabasis.epub', 'one.xhtml', '12 34 Anabasis.epub')).toBe(
      '<p>Up country</p>',
    )
    expect(asked.readBookMarkup).toHaveBeenCalledWith({
      path: 'Anabasis.epub',
      document: 'one.xhtml',
      seen: { path: 'Anabasis.epub', size: 12n, mtime: 34n },
    })
  })

  it('names no file where the window read none', async () => {
    asked.readBookMarkup.mockResolvedValue({ markup: '' })

    await books.readMarkup('Anabasis.epub', 'one.xhtml', '')

    expect(asked.readBookMarkup).toHaveBeenCalledWith({
      path: 'Anabasis.epub',
      document: 'one.xhtml',
    })
  })
})

describe('the address an entry of the archive is drawn from', () => {
  it('writes each part of the name out and says which bytes it is about', () => {
    expect(
      books.getEntryUrl('Books/Anabasis.epub', 'OEBPS/images/map 1.png', '12 34 Anabasis.epub'),
    ).toBe('/assets/Books%2FAnabasis.epub/OEBPS/images/map%201.png?size=12&mtime=34')
  })
})
