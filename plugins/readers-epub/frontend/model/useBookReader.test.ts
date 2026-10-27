/**
 * One book as its tab reads it.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import {
  getContents,
  getDocumentAtOffset,
  useBookReader as openBookReader,
  getPageNumber,
  type Book,
  type Books,
} from './useBookReader'
import type { MessageWriter } from './useBookDocument'

const WORDS = { page: 'Page' }

let recordedEvents: string[] = []
const writeMessage = (text: string) => recordedEvents.push(text)

const useBookReader = (
  books: Books,
  path: string,
  words = WORDS,
  writer: MessageWriter = writeMessage,
) => openBookReader(books, path, { words, writeMessage: writer })

beforeEach(() => {
  recordedEvents = []
})

const BOOK: Book = {
  title: 'Mahābhārata',
  span: { start: 0, end: 9_000 },
  documents: [
    { path: 'text/part0001.xhtml', span: { start: 0, end: 3_000 } },
    { path: 'text/part0002.xhtml', span: { start: 3_000, end: 6_000 } },
    { path: 'text/part0003.xhtml', span: { start: 6_000, end: 9_000 } },
  ],
  parts: [
    { title: 'Ādi Parva', offset: 0, level: 0 },
    { title: 'Сказание о сожжении леса', offset: 3_600, level: 1 },
  ],
  printedPages: [],
  pages: 3,
  pageBytes: 3_000,
  stat: '20480 1700000000000000000 mahabharata.epub',
}

const UNNAMED: Book = { ...BOOK, parts: [] }

const PRINTED: Book = {
  ...UNNAMED,
  printedPages: [
    { label: 'i', offset: 0 },
    { label: '1', offset: 3_600 },
  ],
}

function shelf(book: Book | Error = BOOK, markup: Error | null = null) {
  const requests: string[] = []
  const markupCalls: string[] = []
  const placeCalls: string[] = []

  const books: Books = {
    getBook: async (path) => {
      requests.push(path)
      if (book instanceof Error) throw book
      return book
    },
    readMarkup: async (path, document, stat) => {
      markupCalls.push(`${path} ${document} ${stat}`)
      if (markup) throw markup
      return `<p data-offset="0">${document}</p>`
    },
    getEntryUrl: (path, name) => `/assets/${path}/${name}`,
    getCfi: async (path, offset) => {
      placeCalls.push(`cfi ${path} ${offset}`)
      return `epubcfi(/8/2!/4/2/1:${offset})`
    },
    findOffset: async (path, cfi) => {
      placeCalls.push(`offset ${path} ${cfi}`)
      return cfi === 'epubcfi(/8/4!/4/4/1:600)' ? 3_600 : null
    },
  }

  return { books, requests, markupCalls, placeCalls }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

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

  it('is the pages it was printed on, where it is empty', () => {
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

describe('a book still being read', () => {
  it('says so until the book lands, and no longer', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    expect(readItem.isLoading.value).toBe(true)

    await readItem.goToOffset(0)

    expect(readItem.isLoading.value).toBe(false)
  })

  it('stops saying so even where the book could not be read', async () => {
    const { books } = shelf(new Error('no such book'))
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(0)

    expect(readItem.isLoading.value).toBe(false)
  })
})

describe('a book opened', () => {
  it('asks what it is once, however much is read of it', async () => {
    const { books, requests } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(3_000)
    await readItem.goToOffset(0)

    expect(requests).toStrictEqual(['library/Mahabharata.epub'])
  })

  it('remains at the first byte of its text, in the document that holds it', async () => {
    const { books, markupCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await settle()

    expect(readItem.offset.value).toBe(0)
    expect(readItem.pages.value).toBe(3)
    expect(readItem.page.value).toBe(1)
    expect(readItem.documentSpan.value).toStrictEqual({ start: 0, end: 3_000 })
    expect(markupCalls).toStrictEqual([
      'library/Mahabharata.epub text/part0001.xhtml 20480 1700000000000000000 mahabharata.epub',
    ])
  })

  it('goes to the offset a CFI names', async () => {
    const { books, placeCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    expect(await readItem.goToCfi('epubcfi(/8/4!/4/4/1:600)')).toBe(true)

    expect(readItem.offset.value).toBe(3_600)
    expect(placeCalls).toStrictEqual(['offset library/Mahabharata.epub epubcfi(/8/4!/4/4/1:600)'])
  })

  it('stays where it is for a CFI that names no place in the book', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    expect(await readItem.goToCfi('epubcfi(/8/10!/4/2)')).toBe(false)

    expect(readItem.offset.value).toBe(0)
  })

  it('asks the backend for the CFI of an offset of its book', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    expect(await readItem.getCfi(42)).toBe('epubcfi(/8/2!/4/2/1:42)')
  })

  it('is called what the book calls itself', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await settle()

    expect(readItem.title.value).toBe('Mahābhārata')
  })
})

describe('where the person is standing', () => {
  it('opens the document the offset falls in, and draws it once', async () => {
    const { books, markupCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(6_500)
    await readItem.goToOffset(6_600)

    expect(readItem.documentSpan.value).toStrictEqual({ start: 6_000, end: 9_000 })
    expect(markupCalls).toHaveLength(2)
    expect(markupCalls[1]).toContain('text/part0003.xhtml')
  })

  it('is the page that offset falls on', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(6_144)

    expect(readItem.page.value).toBe(3)
  })

  it('is never past either end of the book', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(-40)
    expect(readItem.offset.value).toBe(0)

    await readItem.goToOffset(9_000)
    expect(readItem.offset.value).toBe(8_999)
  })

  it('draws nothing again where the offset stays in the document on screen', async () => {
    const { books, markupCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(100)
    await readItem.goToOffset(2_900)

    expect(markupCalls).toHaveLength(1)
  })
})

describe('a passage reached', () => {
  it('marks the run it was sent to and remains there', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.focusSpans({ start: 3_600, end: 3_642 })

    expect(readItem.offset.value).toBe(3_600)
    expect(readItem.documentSpan.value).toStrictEqual({ start: 3_000, end: 6_000 })
    expect(readItem.highlights.value).toStrictEqual([{ start: 3_600, end: 3_642 }])
  })

  it('leaves the other spans somewhere else to look', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.focusSpans({ start: 100, end: 110 }, { start: 6_500, end: 6_520 })

    expect(readItem.offset.value).toBe(100)
    expect(readItem.otherHighlights.value).toStrictEqual([{ start: 6_500, end: 6_520 }])
  })

  it('is nowhere at all when nothing was requested about', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.focusSpans()

    expect(readItem.highlights.value).toStrictEqual([])
  })
})

describe('a book that will not open', () => {
  it('says what went wrong and remains with nothing in it', async () => {
    const { books } = shelf(new Error('this file is not a book'))
    const readItem = useBookReader(books, 'library/Broken.epub', WORDS, writeMessage)

    await readItem.goToOffset(400)

    expect(recordedEvents.join(' ')).toContain('Numen is not responding')
    expect(recordedEvents.join(' ')).not.toContain('this file is not a book')
    expect(readItem.pages.value).toBe(0)
    expect(readItem.markup.value).toBe('')
  })

  it('says what went wrong where a document of it will not be drawn', async () => {
    const { books } = shelf(BOOK, new Error('this document is not in the book'))
    useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await settle()

    expect(recordedEvents.join(' ')).toContain('Numen is not responding')
  })
})

describe('a book tab closed', () => {
  it('asks for nothing more and draws nothing', async () => {
    const { books, markupCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)
    await settle()

    readItem.close()
    await readItem.goToOffset(6_500)

    expect(markupCalls).toHaveLength(1)
    expect(readItem.markup.value).toBe('')
  })
})

describe('a link inside a book followed', () => {
  it('draws the document it names, from the beginning of that document', async () => {
    const { books } = shelf()
    const book = useBookReader(books, 'library/mbh.epub', WORDS, writeMessage)
    await settle()

    await book.followLink('text/part0003.xhtml')

    expect(book.offset.value).toBe(6_000)
    expect(book.renderedChapter.value).toBe('text/part0003.xhtml')
  })

  it('leads nowhere, where the book holds no such document', async () => {
    const { books } = shelf()
    const book = useBookReader(books, 'library/mbh.epub', WORDS, writeMessage)
    await settle()

    await book.followLink('text/nowhere.xhtml')

    expect(book.offset.value).toBe(0)
    expect(book.renderedChapter.value).toBe('text/part0001.xhtml')
  })
})

describe('a book before it lands', () => {
  it('is untitled, spans nothing, and holds no contents', () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    expect(readItem.title.value).toBe('')
    expect(readItem.span.value).toStrictEqual({ start: 0, end: 0 })
    expect(readItem.contents.value).toStrictEqual([])
  })
})

describe('the chapter the person is in', () => {
  it('is the last place of the contents beginning at or before the offset', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(3_599)
    expect(readItem.chapter.value).toBe('Ādi Parva')

    await readItem.goToOffset(3_600)
    expect(readItem.chapter.value).toBe('Сказание о сожжении леса')

    await readItem.goToOffset(0)
    expect(readItem.chapter.value).toBe('Ādi Parva')
  })

  it('is none where the offset remains before the first place of the contents', async () => {
    const { books } = shelf({ ...BOOK, parts: [{ title: 'Late', offset: 100, level: 0 }] })
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    await readItem.goToOffset(0)

    expect(readItem.chapter.value).toBe('')
  })
})

describe('a book that holds no document', () => {
  it('stays where it began wherever it is sent', async () => {
    const { books, markupCalls } = shelf({ ...BOOK, documents: [] })
    const readItem = useBookReader(books, 'library/Empty.epub', WORDS, writeMessage)

    await readItem.goToOffset(500)

    expect(readItem.offset.value).toBe(0)
    expect(markupCalls).toStrictEqual([])
  })
})

describe('a book tab closed before the book lands', () => {
  it('takes nothing of the book and stops saying it is loading', async () => {
    const { books, markupCalls } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)

    readItem.close()
    await settle()

    expect(readItem.title.value).toBe('')
    expect(readItem.pages.value).toBe(0)
    expect(readItem.contents.value).toStrictEqual([])
    expect(markupCalls).toStrictEqual([])
    expect(readItem.isLoading.value).toBe(false)
  })

  it('says nothing of a book that could not be read', async () => {
    const { books } = shelf(new Error('no such book'))
    const readItem = useBookReader(books, 'library/Broken.epub', WORDS, writeMessage)

    readItem.close()
    await settle()

    expect(recordedEvents).toStrictEqual([])
  })
})

describe('a book tab closed after the book landed', () => {
  it('lets go of its contents and stays where it stood', async () => {
    const { books } = shelf()
    const readItem = useBookReader(books, 'library/Mahabharata.epub', WORDS, writeMessage)
    await settle()
    expect(readItem.contents.value).toHaveLength(2)

    readItem.close()
    await readItem.goToOffset(6_500)

    expect(readItem.contents.value).toStrictEqual([])
    expect(readItem.offset.value).toBe(0)
  })
})

describe('what is returned of a book that will not open', () => {
  it('is returned as an error', async () => {
    const { books } = shelf(new Error('no such book'))
    const response: Array<string | undefined> = []
    const readItem = useBookReader(books, 'library/Broken.epub', WORDS, (_text, level) =>
      response.push(level),
    )

    await readItem.goToOffset(0)

    expect(response).toStrictEqual(['error'])
  })
})
