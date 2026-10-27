/**
 * The spine of a book and the markup of one document of it.
 *
 * What is proved here is the crossing: a document of the spine arrives as the
 * range of the text it covers, and the file the caller read goes back out as
 * the three parts the schema holds it in, or not at all.
 */
import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it, vi } from 'vitest'
import { books, client } from './wire'

const request = {
  getBook: vi.fn(),
  readBookMarkup: vi.fn(),
  getCfi: vi.fn(),
  findOffset: vi.fn(),
}
client.books = request as unknown as typeof client.books

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
  it('arrives back as its spine, its parts and the file it was read from', async () => {
    request.getBook.mockResolvedValue({
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
    expect(request.getBook).toHaveBeenCalledWith({ path: 'Anabasis.epub' })
  })

  it('names no file where the answer carries none', async () => {
    request.getBook.mockResolvedValue(nothing)

    expect((await books.getBook('Anabasis.epub')).fingerprint).toBe('')
  })
})

describe('the markup of one document', () => {
  it('names the file the window read', async () => {
    request.readBookMarkup.mockResolvedValue({ markup: '<p>Up country</p>' })

    expect(await books.readMarkup('Anabasis.epub', 'one.xhtml', '12 34 Anabasis.epub')).toBe(
      '<p>Up country</p>',
    )
    expect(request.readBookMarkup).toHaveBeenCalledWith({
      path: 'Anabasis.epub',
      document: 'one.xhtml',
      baseline: { path: 'Anabasis.epub', size: 12n, mtime: 34n },
    })
  })

  it('names no file where the window read none', async () => {
    request.readBookMarkup.mockResolvedValue({ markup: '' })

    await books.readMarkup('Anabasis.epub', 'one.xhtml', '')

    expect(request.readBookMarkup).toHaveBeenCalledWith({
      path: 'Anabasis.epub',
      document: 'one.xhtml',
    })
  })
})

describe('a place in a book', () => {
  it('is the CFI the backend names an offset by', async () => {
    request.getCfi.mockResolvedValue({ cfi: 'epubcfi(/8/4!/4/2/1:17)' })

    expect(await books.getCfi('Anabasis.epub', 120)).toBe('epubcfi(/8/4!/4/2/1:17)')
    expect(request.getCfi).toHaveBeenCalledWith({ path: 'Anabasis.epub', offset: 120 })
  })

  it('is the offset the backend finds a CFI at', async () => {
    request.findOffset.mockResolvedValue({ offset: 120 })

    expect(await books.findOffset('Anabasis.epub', 'epubcfi(/8/4!/4/2/1:17)')).toBe(120)
    expect(request.findOffset).toHaveBeenCalledWith({
      path: 'Anabasis.epub',
      cfi: 'epubcfi(/8/4!/4/2/1:17)',
    })
  })

  it('is nothing where the backend finds no such place or reads no CFI', async () => {
    request.findOffset.mockRejectedValueOnce(new ConnectError('no document', Code.NotFound))
    request.getCfi.mockRejectedValueOnce(new ConnectError('no text', Code.NotFound))
    request.findOffset.mockRejectedValueOnce(new ConnectError('no CFI', Code.InvalidArgument))

    expect(await books.findOffset('Anabasis.epub', 'epubcfi(/6/2!/4)')).toBeNull()
    expect(await books.getCfi('Anabasis.epub', 0)).toBeNull()
    expect(await books.findOffset('Anabasis.epub', 'page 3')).toBeNull()
  })

  it('passes on any other failure', async () => {
    request.getCfi.mockRejectedValueOnce(new ConnectError('disk', Code.Internal))

    await expect(books.getCfi('Anabasis.epub', 0)).rejects.toThrow('disk')
  })
})

describe('the address an entry of the archive is drawn from', () => {
  it('writes each part of the name out and says which bytes it is about', () => {
    expect(
      books.getEntryUrl('Books/Anabasis.epub', 'OEBPS/images/map 1.png', '12 34 Anabasis.epub'),
    ).toBe('/assets/Books%2FAnabasis.epub/OEBPS/images/map%201.png?size=12&mtime=34')
  })
})

describe('an entry of a book', () => {
  it('is served on the plugin route, with the book and the file it was read from', () => {
    expect(
      books.getEntryUrl('Books/Anabasis.epub', 'OEBPS/images/map one.png', '12 34 Anabasis.epub'),
    ).toBe(
      '/plugin/md.numen.readers.epub/entries/OEBPS/images/map%20one.png?path=Books%2FAnabasis.epub&size=12&mtime=34',
    )
  })

  it('names no file where the book was read from none', () => {
    expect(books.getEntryUrl('Anabasis.epub', 'cover.png', '')).toBe(
      '/plugin/md.numen.readers.epub/entries/cover.png?path=Anabasis.epub',
    )
  })
})
