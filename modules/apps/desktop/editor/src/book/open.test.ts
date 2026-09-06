/**
 * One book as its tab reads it.
 *
 * A place in a book is a byte offset, so which document that offset falls in,
 * which page it falls on, what the contents list holds and what a search marks
 * are decisions, and they are the ones asked about here.
 */
import { describe, expect, it } from 'vitest'
import { contentsOf, documentAt, openBook, pageAt, type Book, type Books } from './open'

/** What a book tab says, as far as the state says anything. */
const WORDS = { page: 'Page' }

/**
 * A book of three documents, its text counted in bytes of Devanagari and
 * Cyrillic: three bytes a letter and two, so no offset here is a count of
 * characters.
 */
const BOOK: Book = {
  title: 'Mahābhārata',
  span: { begins: 0, ends: 9_000 },
  documents: [
    { path: 'text/part0001.xhtml', span: { begins: 0, ends: 3_000 } },
    { path: 'text/part0002.xhtml', span: { begins: 3_000, ends: 6_000 } },
    { path: 'text/part0003.xhtml', span: { begins: 6_000, ends: 9_000 } },
  ],
  parts: [
    { title: 'Ādi Parva', at: 0, level: 0 },
    { title: 'Сказание о сожжении леса', at: 3_600, level: 1 },
  ],
  printed: [],
  pages: 3,
  at: '20480 1700000000000000000 mahabharata.epub',
}

/** The same book, naming nothing at all, which half of this corpus does. */
const UNNAMED: Book = { ...BOOK, parts: [] }

/** The same again, printed on pages a person can be sent to. */
const PRINTED: Book = {
  ...UNNAMED,
  printed: [
    { label: 'i', at: 0 },
    { label: '1', at: 3_600 },
  ],
}

/**
 * A book on a shelf, recording every question put to it. Each document is drawn
 * as markup naming itself.
 */
function shelf(book: Book | Error = BOOK, markup: Error | null = null) {
  const asked: string[] = []
  /** Every document it was asked to draw. */
  const drawn: string[] = []

  const books: Books = {
    shape: async (path) => {
      asked.push(path)
      if (book instanceof Error) throw book
      return book
    },
    markup: async (path, document, seen) => {
      drawn.push(`${path} ${document} ${seen}`)
      if (markup) throw markup
      return `<p data-offset="0">${document}</p>`
    },
    entry: (path, name) => `/assets/${path}/entries/${name}`,
  }

  return { books, asked, drawn }
}

/** A moment for what the book was asked to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

describe('the page an offset falls on', () => {
  it('is where the offset stands along the text, counted from one', () => {
    const span = { begins: 0, ends: 9_000 }

    expect(pageAt(span, 3, 0)).toBe(1)
    expect(pageAt(span, 3, 2_999)).toBe(1)
    expect(pageAt(span, 3, 3_000)).toBe(2)
    expect(pageAt(span, 3, 8_999)).toBe(3)
  })

  it('is never past the last page', () => {
    expect(pageAt({ begins: 0, ends: 9_000 }, 3, 900_000)).toBe(3)
  })

  it('is no page at all for a book with none', () => {
    expect(pageAt({ begins: 0, ends: 0 }, 0, 0)).toBe(0)
  })
})

describe('the document an offset falls in', () => {
  it('is the last one beginning at or before it', () => {
    expect(documentAt(BOOK.documents, 3_000)?.path).toBe('text/part0002.xhtml')
    expect(documentAt(BOOK.documents, 2_999)?.path).toBe('text/part0001.xhtml')
    expect(documentAt(BOOK.documents, 8_999)?.path).toBe('text/part0003.xhtml')
  })

  it('is no document where the book has none there', () => {
    expect(documentAt([], 0)).toBeUndefined()
  })
})

describe('what a book is reached by', () => {
  it('is the places it names', () => {
    expect(contentsOf(BOOK, WORDS)).toStrictEqual([
      { title: 'Ādi Parva', at: 0, level: 0 },
      { title: 'Сказание о сожжении леса', at: 3_600, level: 1 },
    ])
  })

  it('is the pages it was printed on, where it names nothing', () => {
    expect(contentsOf(PRINTED, WORDS)).toStrictEqual([
      { title: 'Page i', at: 0, level: 0 },
      { title: 'Page 1', at: 3_600, level: 0 },
    ])
  })

  it('is the documents it is read in, where it was printed on nothing', () => {
    // Forty books of the Mahābhārata name no part and carry no printed page,
    // and a book of five thousand pages with nothing to jump by is one nobody
    // finds their way back into.
    expect(contentsOf(UNNAMED, WORDS)).toStrictEqual([
      { title: 'part0001', at: 0, level: 0 },
      { title: 'part0002', at: 3_000, level: 0 },
      { title: 'part0003', at: 6_000, level: 0 },
    ])
  })
})

describe('a book opened', () => {
  it('asks what it is once, however much is read of it', async () => {
    const { books, asked } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.go(3_000)
    await read.go(0)

    expect(asked).toStrictEqual(['library/Mahabharata.epub'])
  })

  it('stands at the first byte of its text, in the document that holds it', async () => {
    const { books, drawn } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await settles()

    expect(read.at.value).toBe(0)
    expect(read.pages.value).toBe(3)
    expect(read.page.value).toBe(1)
    expect(read.reading.value).toStrictEqual({ begins: 0, ends: 3_000 })
    expect(drawn).toStrictEqual([
      'library/Mahabharata.epub text/part0001.xhtml 20480 1700000000000000000 mahabharata.epub',
    ])
  })

  it('is called what the book calls itself', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await settles()

    expect(read.title.value).toBe('Mahābhārata')
  })
})

describe('where the person is standing', () => {
  it('opens the document the offset falls in, and draws it once', async () => {
    const { books, drawn } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.go(6_500)
    await read.go(6_600)

    expect(read.reading.value).toStrictEqual({ begins: 6_000, ends: 9_000 })
    expect(drawn).toHaveLength(2)
    expect(drawn[1]).toContain('text/part0003.xhtml')
  })

  it('is the page that offset falls on', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.go(6_144)

    expect(read.page.value).toBe(3)
  })

  it('is never past either end of the book', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.go(-40)
    expect(read.at.value).toBe(0)

    // Turning on past the last document asks for the byte the book ends at,
    // which is one byte past its last.
    await read.go(9_000)
    expect(read.at.value).toBe(8_999)
  })

  it('draws nothing again where the offset stays in the document on screen', async () => {
    const { books, drawn } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.go(100)
    await read.go(2_900)

    expect(drawn).toHaveLength(1)
  })
})

describe('a passage reached', () => {
  it('marks the run it was sent to and stands there', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.reach({ start: 3_600, length: 42 })

    expect(read.at.value).toBe(3_600)
    expect(read.reading.value).toStrictEqual({ begins: 3_000, ends: 6_000 })
    expect(read.marked.value).toStrictEqual([{ begins: 3_600, ends: 3_642 }])
  })

  it('leaves the other stretches somewhere else to look', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.reach({ start: 100, length: 10 }, { start: 6_500, length: 20 })

    expect(read.at.value).toBe(100)
    expect(read.also.value).toStrictEqual([{ begins: 6_500, ends: 6_520 }])
  })

  it('is nowhere at all when nothing was asked about', async () => {
    const { books } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await read.reach()

    expect(read.marked.value).toStrictEqual([])
  })
})

describe('a book that will not open', () => {
  it('says what went wrong and stands with nothing in it', async () => {
    const { books } = shelf(new Error('this file is not a book'))
    const read = openBook(books, 'library/Broken.epub', WORDS)

    await read.go(400)

    expect(read.trouble.value).toContain('numen did not answer')
    expect(read.trouble.value).not.toContain('this file is not a book')
    expect(read.pages.value).toBe(0)
    expect(read.markup.value).toBe('')
  })

  it('says what went wrong where a document of it will not be drawn', async () => {
    const { books } = shelf(BOOK, new Error('this document is not in the book'))
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)

    await settles()

    expect(read.trouble.value).toContain('numen did not answer')
  })
})

describe('a book tab closed', () => {
  it('asks for nothing more and draws nothing', async () => {
    const { books, drawn } = shelf()
    const read = openBook(books, 'library/Mahabharata.epub', WORDS)
    await settles()

    read.close()
    await read.go(6_500)

    expect(drawn).toHaveLength(1)
    expect(read.markup.value).toBe('')
  })
})
