/**
 * One document as its tab reads it.
 *
 * A page is a picture the application draws to a width, so which page is asked
 * for, at what width, and when the same one is asked for again are decisions,
 * and they are the ones asked about here.
 */
import { describe, expect, it } from 'vitest'
import { useDocumentReader, type Documents, type PageHighlight, type DocumentLayout } from './open'

const LAYOUT: DocumentLayout = {
  pages: [
    { width: 612, height: 792 },
    { width: 612, height: 792 },
    { width: 612, height: 792 },
  ],
  fingerprint: '1024 1700000000000000000 book.pdf',
}

/**
 * A document of three pages, recording every question put to it. It answers
 * about each span asked about with the highlights standing at the same place
 * in `where`, and with nothing where that list is shorter.
 */
function book(layout: DocumentLayout | Error = LAYOUT, where: readonly (readonly PageHighlight[])[] = []) {
  const asked: string[] = []
  /** Every span of the document's text it was asked what stands on. */
  const spans: string[] = []

  const documents: Documents = {
    getDocumentLayout: async (path) => {
      asked.push(path)
      if (layout instanceof Error) throw layout
      return layout
    },
    getPageUrl: (path, at, wide) => `${path} ${at} ${wide}`,
    getHighlights: async (path, asking) => {
      for (const one of asking) spans.push(`${path} ${one.from} ${one.to}`)
      return asking.map((_, i) => where[i] ?? [])
    },
  }

  return { documents, asked, spans }
}

describe('a document opened', () => {
  it('asks what it is once, however much is read of it', async () => {
    const { documents, asked } = book()
    const read = useDocumentReader(documents, 'Book.pdf')

    read.widen(800)
    await read.next()
    await read.back()

    expect(asked).toStrictEqual(['Book.pdf'])
  })

  it('stands on the first page, under the name the document gives it', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')

    await read.go(0)

    expect(read.pages.value.length).toBe(3)
    expect(read.at.value).toBe(0)
  })

  it('draws nothing until it is told how wide the page is', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')

    await read.go(1)

    expect(read.picture.value).toBe('')
  })
})

describe('a page turned', () => {
  it('is drawn at the page it moved to', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)
    await read.go(0)

    await read.next()

    expect(read.at.value).toBe(1)
    expect(read.picture.value).toBe('Book.pdf 1 800')
  })

  it('is the picture it already was when it is turned back to', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)
    await read.go(0)
    const first = read.picture.value

    await read.go(1)
    await read.go(0)

    expect(read.picture.value).toBe(first)
  })

  it('stands at the end when it is turned past it', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)

    await read.go(9)
    expect(read.at.value).toBe(2)

    await read.go(-4)
    expect(read.at.value).toBe(0)
  })
})

describe('the width a page is drawn at', () => {
  it('is the width the page is asked for', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    await read.go(0)

    read.widen(800)
    expect(read.picture.value).toBe('Book.pdf 0 800')

    read.widen(1600)
    expect(read.picture.value).toBe('Book.pdf 0 1600')
  })

  it('is never wider than a page is drawn', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    await read.go(0)

    read.widen(9000)

    expect(read.picture.value).toBe('Book.pdf 0 4096')
  })
})

describe('what is highlighted', () => {
  it('is the rectangles of the page in front, and the tab turns to the first', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)
    const rect = { minX: 0.1, minY: 0.2, maxX: 0.9, maxY: 0.3 }

    await read.highlight([[{ page: 2, rects: [rect] }]])

    expect(read.at.value).toBe(2)
    expect(read.highlighted.value).toStrictEqual([rect])

    await read.go(0)
    expect(read.highlighted.value).toStrictEqual([])
  })
})

describe('a document opened at a place in its text', () => {
  it('highlights what stands there, on the first page it falls on', async () => {
    const rect = { minX: 0.1, minY: 0.2, maxX: 0.4, maxY: 0.23 }
    const { documents, spans } = book(LAYOUT, [[{ page: 1, rects: [rect] }]])
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)

    await read.reach({ from: 40_512, to: 40_543 })

    expect(spans).toStrictEqual(['Book.pdf 40512 40543'])
    expect(read.at.value).toBe(1)
    expect(read.highlighted.value).toStrictEqual([rect])
  })

  it('highlights the other places asked for where they fall, apart from the first', async () => {
    const here = { minX: 0.1, minY: 0.2, maxX: 0.4, maxY: 0.23 }
    const there = { minX: 0.1, minY: 0.5, maxX: 0.4, maxY: 0.53 }
    const alsoThere = { minX: 0.1, minY: 0.8, maxX: 0.4, maxY: 0.83 }
    const { documents, spans } = book(LAYOUT, [
      [{ page: 1, rects: [here] }],
      [{ page: 1, rects: [there] }],
      [{ page: 2, rects: [alsoThere] }],
    ])
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)

    await read.reach(
      { from: 40_512, to: 40_543 },
      { from: 41_000, to: 41_020 },
      { from: 90_000, to: 90_012 },
    )

    expect(spans).toStrictEqual([
      'Book.pdf 40512 40543',
      'Book.pdf 41000 41020',
      'Book.pdf 90000 90012',
    ])
    // The tab stands at the first place, which is the one highlighted.
    expect(read.at.value).toBe(1)
    expect(read.highlighted.value).toStrictEqual([here])
    expect(read.also.value).toStrictEqual([there])
    // The third place falls on another page and is highlighted there.
    expect(read.alsoOn(2)).toStrictEqual([alsoThere])
    expect(read.highlightedOn(2)).toStrictEqual([])
  })

  it('stands on the first page with nothing highlighted where nothing stands there', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)

    await read.reach({ from: 40_512, to: 40_543 })

    expect(read.at.value).toBe(0)
    expect(read.highlighted.value).toStrictEqual([])
    expect(read.error.value).toBe('')
  })

  it('says what it could not ask, and reads on', async () => {
    const { documents } = book()
    documents.getHighlights = async () => {
      throw new Error('the layer is being written')
    }
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)

    await read.reach({ from: 40_512, to: 40_543 })

    expect(read.error.value).toContain('numen did not answer')
    expect(read.error.value).not.toContain('the layer is being written')
    expect(read.picture.value).toBe('Book.pdf 0 800')
  })
})

describe('a document tab that closes', () => {
  it('draws nothing more, and turns to no other page', async () => {
    const { documents } = book()
    const read = useDocumentReader(documents, 'Book.pdf')
    read.widen(800)
    await read.go(1)

    read.close()
    await read.go(2)

    expect(read.picture.value).toBe('')
    expect(read.at.value).toBe(1)
  })
})

describe('a document that will not open', () => {
  it('says why, and stands with nothing in it', async () => {
    const { documents } = book(new Error('no such document'))
    const read = useDocumentReader(documents, 'Gone.pdf')

    read.widen(800)
    await read.go(1)

    expect(read.error.value).toContain('numen did not answer')
    expect(read.error.value).not.toContain('no such document')
    expect(read.pages.value.length).toBe(0)
    expect(read.picture.value).toBe('')
  })
})
