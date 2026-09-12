/**
 * One book as its tab reads it.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { getContents, getDocumentAtOffset, useBookReader, getPageNumber, type Book, type Books } from './useBookReader'

const WORDS = { page: 'Page' }

let heard: string[] = []
const said = (text: string) => heard.push(text)

beforeEach(() => {
  heard = []
})

const BOOK: Book = {
  title: 'Mahābhārata',
  span: { begins: 0, ends: 9_000 },
  documents: [
    { path: 'text/part0001.xhtml', span: { begins: 0, ends: 3_000 } },
    { path: 'text/part0002.xhtml', span: { begins: 3_000, ends: 6_000 } },
    { path: 'text/part0003.xhtml', span: { begins: 6_000, ends: 9_000 } },
  ],
  parts: [
    { title: 'Ādi Parva', offset: 0, level: 0 },
    { title: 'Сказание о сожжении леса', offset: 3_600, level: 1 },
  ],
  printed: [],
  pages: 3,
  pageBytes: 3_000,
  fingerprint: '20480 1700000000000000000 mahabharata.epub',
}

const UNNAMED: Book = { ...BOOK, parts: [] }

const PRINTED: Book = {
  ...UNNAMED,
  printed: [
    { label: 'i', offset: 0 },
    { label: '1', offset: 3_600 },
  ],
}

function shelf(book: Book | Error = BOOK, markup: Error | null = null) {
  const asked: string[] = []
  const drawn: string[] = []

  const books: Books = {
    getBook: async (path) => {
      asked.push(path)
      if (book instanceof Error) throw book
      return book
    },
    readMarkup: async (path, document, fingerprint) => {
      drawn.push(`${path} ${document} ${fingerprint}`)
      if (markup) throw markup
      return `<p data-offset="0">${document}</p>`
    },
    getEntryUrl: (path, name) => `/assets/${path}/${name}`,
  }

  return { books, asked, drawn }
}

const settles = () => new Promise((done) => setTimeout(done, 0))

describe('the page an offset falls on', () => {
  it('is the offset over the bytes a page of this book holds, counted from one', () => {
    expect(getPageNumber(3_000, 3, 0)).toBe(1)
    expect(getPageNumber(3_000, 3, 2_999)).toBe(1)
    expect(getPageNumber(3_000, 3, 3_000)).toBe(2)
    expect(getPageNumber(3_000, 3, 8_999)).toBe(3)
  })

  it('is never past the last page', () => {
    expect(getPageNumber(3_000, 3, 900_000)).toBe(3)
  })

  it('is no page at all for a book with none', () => {
    expect(getPageNumber(0, 0, 0)).toBe(0)
    expect(getPageNumber(0, 3, 100)).toBe(0)
  })
})

describe('the document an offset falls in', () => {
  it('is the last one beginning at or before it', () => {
    expect(getDocumentAtOffset(BOOK.documents, 3_000)?.path).toBe('text/part0002.xhtml')
    expect(getDocumentAtOffset(BOOK.documents, 2_999)?.path).toBe('text/part0001.xhtml')
    expect(getDocumentAtOffset(BOOK.documents, 8_999)?.path).toBe('text/part0003.xhtml')
  })

  it('is no document where the book has none there', () => {
    expect(getDocumentAtOffset([], 0)).toBeUndefined()
  })
})

describe('what a book is reached by', () => {
  it('is the places it names', () => {
    expect(getContents(BOOK, WORDS)).toStrictEqual([
      { title: 'Ādi Parva', at: 0, level: 0 },
      { title: 'Сказание о сожжении леса', at: 3_600, level: 1 },
    ])
  })

  it('is the pages it was printed on, where it names nothing', () => {
    expect(getContents(PRINTED, WORDS)).toStrictEqual([
      { title: 'Page i', at: 0, level: 0 },
      { title: 'Page 1', at: 3_600, level: 0 },
    ])
  })

  it('is the documents it is read in, where it was printed on nothing', () => {
    expect(getContents(UNNAMED, WORDS)).toStrictEqual([
      { title: 'part0001', at: 0, level: 0 },
      { title: 'part0002', at: 3_000, level: 0 },
      { title: 'part0003', at: 6_000, level: 0 },
    ])
  })
})

describe('a book opened', () => {
  it('asks what it is once, however much is read of it', async () => {
    const { books, asked } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.goToOffset(3_000)
    await read.goToOffset(0)

    expect(asked).toStrictEqual(['library/Mahabharata.epub'])
  })

  it('stands at the first byte of its text, in the document that holds it', async () => {
    const { books, drawn } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await settles()

    expect(read.offset.value).toBe(0)
    expect(read.pages.value).toBe(3)
    expect(read.page.value).toBe(1)
    expect(read.reading.value).toStrictEqual({ begins: 0, ends: 3_000 })
    expect(drawn).toStrictEqual([
      'library/Mahabharata.epub text/part0001.xhtml 20480 1700000000000000000 mahabharata.epub',
    ])
  })

  it('is called what the book calls itself', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await settles()

    expect(read.title.value).toBe('Mahābhārata')
  })
})

describe('where the person is standing', () => {
  it('opens the document the offset falls in, and draws it once', async () => {
    const { books, drawn } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.goToOffset(6_500)
    await read.goToOffset(6_600)

    expect(read.reading.value).toStrictEqual({ begins: 6_000, ends: 9_000 })
    expect(drawn).toHaveLength(2)
    expect(drawn[1]).toContain('text/part0003.xhtml')
  })

  it('is the page that offset falls on', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.goToOffset(6_144)

    expect(read.page.value).toBe(3)
  })

  it('is never past either end of the book', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.goToOffset(-40)
    expect(read.offset.value).toBe(0)

    await read.goToOffset(9_000)
    expect(read.offset.value).toBe(8_999)
  })

  it('draws nothing again where the offset stays in the document on screen', async () => {
    const { books, drawn } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.goToOffset(100)
    await read.goToOffset(2_900)

    expect(drawn).toHaveLength(1)
  })
})

describe('a passage reached', () => {
  it('marks the run it was sent to and stands there', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.focusSpans({ from: 3_600, to: 3_642 })

    expect(read.offset.value).toBe(3_600)
    expect(read.reading.value).toStrictEqual({ begins: 3_000, ends: 6_000 })
    expect(read.highlights.value).toStrictEqual([{ begins: 3_600, ends: 3_642 }])
  })

  it('leaves the other spans somewhere else to look', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.focusSpans({ from: 100, to: 110 }, { from: 6_500, to: 6_520 })

    expect(read.offset.value).toBe(100)
    expect(read.elsewhere.value).toStrictEqual([{ begins: 6_500, ends: 6_520 }])
  })

  it('is nowhere at all when nothing was asked about', async () => {
    const { books } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await read.focusSpans()

    expect(read.highlights.value).toStrictEqual([])
  })
})

describe('a book that will not open', () => {
  it('says what went wrong and stands with nothing in it', async () => {
    const { books } = shelf(new Error('this file is not a book'))
    const read = useBookReader(books, 'library/Broken.epub', WORDS, said)

    await read.goToOffset(400)

    expect(heard.join(' ')).toContain('numen did not answer')
    expect(heard.join(' ')).not.toContain('this file is not a book')
    expect(read.pages.value).toBe(0)
    expect(read.markup.value).toBe('')
  })

  it('says what went wrong where a document of it will not be drawn', async () => {
    const { books } = shelf(BOOK, new Error('this document is not in the book'))
    useBookReader(books, 'library/Mahabharata.epub', WORDS, said)

    await settles()

    expect(heard.join(' ')).toContain('numen did not answer')
  })
})

describe('a book tab closed', () => {
  it('asks for nothing more and draws nothing', async () => {
    const { books, drawn } = shelf()
    const read = useBookReader(books, 'library/Mahabharata.epub', WORDS, said)
    await settles()

    read.close()
    await read.goToOffset(6_500)

    expect(drawn).toHaveLength(1)
    expect(read.markup.value).toBe('')
  })
})

describe('a link inside a book followed', () => {
  it('draws the document it names, from the beginning of that document', async () => {
    const { books } = shelf()
    const book = useBookReader(books, 'library/mbh.epub', WORDS, said)
    await settles()

    await book.followLink('text/part0003.xhtml')

    expect(book.offset.value).toBe(6_000)
    expect(book.drawn.value).toBe('text/part0003.xhtml')
  })

  it('leads nowhere, where the book holds no such document', async () => {
    const { books } = shelf()
    const book = useBookReader(books, 'library/mbh.epub', WORDS, said)
    await settles()

    await book.followLink('text/nowhere.xhtml')

    expect(book.offset.value).toBe(0)
    expect(book.drawn.value).toBe('text/part0001.xhtml')
  })
})
