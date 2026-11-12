/**
 * One document as its tab reads it.
 */
import { describe, expect, it, vi } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createDocuments } from '../api/wire'
import { useDocumentReader } from './useDocumentReader'
import type { Documents, PageHighlight, DocumentLayout } from '../types'

const LAYOUT: DocumentLayout = {
  pages: [
    { width: 612, height: 792 },
    { width: 612, height: 792 },
    { width: 612, height: 792 },
  ],
  stat: '1024 1700000000000000000 book.pdf',
}

function book(
  layout: DocumentLayout | Error = LAYOUT,
  where: readonly (readonly PageHighlight[])[] = [],
) {
  const requests: string[] = []
  const spans: string[] = []

  const documents: Documents = {
    getDocumentLayout: async (path) => {
      requests.push(path)
      if (layout instanceof Error) throw layout
      return layout
    },
    getPageUrl: (path, pageNumber, width) => `${path} ${pageNumber} ${width}`,
    getHighlights: async (path, requestedSpans) => {
      for (const one of requestedSpans) spans.push(`${path} ${one.start} ${one.end}`)
      return requestedSpans.map((_, i) => where[i] ?? [])
    },
  }

  return { documents, requests, spans }
}

describe('a document still being read', () => {
  it('says so until the layout lands, and no longer', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    expect(readItem.state.isLoading.value).toBe(true)

    await readItem.actions.goToPage(0)

    expect(readItem.state.isLoading.value).toBe(false)
  })

  it('stops saying so even where the document could not be read', async () => {
    const { documents } = book(new Error('no such document'))
    const readItem = useDocumentReader(documents, 'Book.pdf')

    await readItem.actions.goToPage(0)

    expect(readItem.state.isLoading.value).toBe(false)
    expect(readItem.state.error.value).not.toBe('')
  })
})

describe('a document opened', () => {
  it('asks what it is once, however much is read of it', async () => {
    const { documents, requests } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    readItem.view.widen(800)
    await readItem.actions.goToNextPage()
    await readItem.actions.goToPreviousPage()

    expect(requests).toStrictEqual(['Book.pdf'])
  })

  it('relies on the first page, under the name the document gives it', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    await readItem.actions.goToPage(0)

    expect(readItem.state.pages.value.length).toBe(3)
    expect(readItem.state.pageNumber.value).toBe(0)
  })

  it('draws nothing until it is configured with how wide the page is', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    await readItem.actions.goToPage(1)

    expect(readItem.view.picture.value).toBe('')
  })
})

describe('a page turned', () => {
  it('is drawn at the page it moved to', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)
    await readItem.actions.goToPage(0)

    await readItem.actions.goToNextPage()

    expect(readItem.state.pageNumber.value).toBe(1)
    expect(readItem.view.picture.value).toBe('Book.pdf 1 800')
  })

  it('is the picture it already was when it is turned back to', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)
    await readItem.actions.goToPage(0)
    const first = readItem.view.picture.value

    await readItem.actions.goToPage(1)
    await readItem.actions.goToPage(0)

    expect(readItem.view.picture.value).toBe(first)
  })

  it('remains at the end when it is turned past it', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)

    await readItem.actions.goToPage(9)
    expect(readItem.state.pageNumber.value).toBe(2)

    await readItem.actions.goToPage(-4)
    expect(readItem.state.pageNumber.value).toBe(0)
  })
})

describe('the width a page is drawn at', () => {
  it('is the width the page is requested for', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    await readItem.actions.goToPage(0)

    readItem.view.widen(800)
    expect(readItem.view.picture.value).toBe('Book.pdf 0 800')

    readItem.view.widen(1600)
    expect(readItem.view.picture.value).toBe('Book.pdf 0 1600')
  })

  it('constrains width to page boundary', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    await readItem.actions.goToPage(0)

    readItem.view.widen(9000)

    expect(readItem.view.picture.value).toBe('Book.pdf 0 4096')
  })
})

describe('what is highlighted', () => {
  it('is the rectangles of the page in front, and the tab turns to the first', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)
    const rect = { minX: 0.1, minY: 0.2, maxX: 0.9, maxY: 0.3 }

    await readItem.highlights.applyHighlights([[{ page: 2, rects: [rect] }]])

    expect(readItem.state.pageNumber.value).toBe(2)
    expect(readItem.highlights.highlighted.value).toStrictEqual([rect])

    await readItem.actions.goToPage(0)
    expect(readItem.highlights.highlighted.value).toStrictEqual([])
  })
})

describe('a document opened at a place in its text', () => {
  it('highlights what remains there, on the first page it falls on', async () => {
    const rect = { minX: 0.1, minY: 0.2, maxX: 0.4, maxY: 0.23 }
    const { documents, spans } = book(LAYOUT, [[{ page: 1, rects: [rect] }]])
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)

    await readItem.actions.focusSpans({ start: 40_512, end: 40_543 })

    expect(spans).toStrictEqual(['Book.pdf 40512 40543'])
    expect(readItem.state.pageNumber.value).toBe(1)
    expect(readItem.highlights.highlighted.value).toStrictEqual([rect])
  })

  it('highlights the other places requested for where they fall, apart from the first', async () => {
    const here = { minX: 0.1, minY: 0.2, maxX: 0.4, maxY: 0.23 }
    const there = { minX: 0.1, minY: 0.5, maxX: 0.4, maxY: 0.53 }
    const onTheNextPage = { minX: 0.1, minY: 0.8, maxX: 0.4, maxY: 0.83 }
    const { documents, spans } = book(LAYOUT, [
      [{ page: 1, rects: [here] }],
      [{ page: 1, rects: [there] }],
      [{ page: 2, rects: [onTheNextPage] }],
    ])
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)

    await readItem.actions.focusSpans(
      { start: 40_512, end: 40_543 },
      { start: 41_000, end: 41_020 },
      { start: 90_000, end: 90_012 },
    )

    expect(spans).toStrictEqual([
      'Book.pdf 40512 40543',
      'Book.pdf 41000 41020',
      'Book.pdf 90000 90012',
    ])
    expect(readItem.state.pageNumber.value).toBe(1)
    expect(readItem.highlights.highlighted.value).toStrictEqual([here])
    expect(readItem.highlights.otherHighlighted.value).toStrictEqual([there])
    expect(readItem.highlights.getOtherHighlightsOn(2)).toStrictEqual([onTheNextPage])
    expect(readItem.highlights.getHighlightsOn(2)).toStrictEqual([])
  })

  it('relies on the first page with nothing highlighted where nothing remains there', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)

    await readItem.actions.focusSpans({ start: 40_512, end: 40_543 })

    expect(readItem.state.pageNumber.value).toBe(0)
    expect(readItem.highlights.highlighted.value).toStrictEqual([])
    expect(readItem.state.error.value).toBe('')
  })

  it('says what it could not ask, and reads on', async () => {
    const { documents } = book()
    documents.getHighlights = async () => {
      throw new Error('the layer is being written')
    }
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)

    await readItem.actions.focusSpans({ start: 40_512, end: 40_543 })

    expect(readItem.state.error.value).toContain('Numen is not responding')
    expect(readItem.state.error.value).not.toContain('the layer is being written')
    expect(readItem.view.picture.value).toBe('Book.pdf 0 800')
  })
})

describe('a document tab that closes', () => {
  it('draws nothing more, and turns to no other page', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    readItem.view.widen(800)
    await readItem.actions.goToPage(1)

    readItem.close()
    await readItem.actions.goToPage(2)

    expect(readItem.view.picture.value).toBe('')
    expect(readItem.state.pageNumber.value).toBe(1)
  })
})

describe('a document that will not open', () => {
  it('says why, and remains with nothing in it', async () => {
    const { documents } = book(new Error('no such document'))
    const readItem = useDocumentReader(documents, 'Gone.pdf')

    readItem.view.widen(800)
    await readItem.actions.goToPage(1)

    expect(readItem.state.error.value).toContain('Numen is not responding')
    expect(readItem.state.error.value).not.toContain('no such document')
    expect(readItem.state.pages.value.length).toBe(0)
    expect(readItem.view.picture.value).toBe('')
  })
})

describe('a page turned back', () => {
  it('is the page before the one in front', async () => {
    const { documents } = book()
    const readItem = useDocumentReader(documents, 'Book.pdf')
    await readItem.actions.goToPage(2)

    await readItem.actions.goToPreviousPage()

    expect(readItem.state.pageNumber.value).toBe(1)
  })
})

describe('a document tab closed before the layout lands', () => {
  const deferLayout = () => {
    let settleLayout: (layout: DocumentLayout | Error) => void = () => undefined
    const pending = new Promise<DocumentLayout>((resolve, reject) => {
      settleLayout = (layout) => (layout instanceof Error ? reject(layout) : resolve(layout))
    })
    const { documents } = book()
    documents.getDocumentLayout = () => pending
    return { documents, settleLayout }
  }

  it('holds no pages when the layout lands', async () => {
    const { documents, settleLayout } = deferLayout()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    readItem.close()
    settleLayout(LAYOUT)
    await readItem.actions.goToPage(0)

    expect(readItem.state.pages.value.length).toBe(0)
    expect(readItem.state.isLoading.value).toBe(false)
  })

  it('says nothing of a layout that could not be read', async () => {
    const { documents, settleLayout } = deferLayout()
    const readItem = useDocumentReader(documents, 'Book.pdf')

    readItem.close()
    settleLayout(new Error('no such document'))
    await readItem.actions.goToPage(0)

    expect(readItem.state.error.value).toBe('')
  })
})

describe('a document tab closed after the layout landed', () => {
  it('asks the document for no highlights', async () => {
    const { documents, spans } = book(LAYOUT, [[{ page: 1, rects: [] }]])
    const readItem = useDocumentReader(documents, 'Book.pdf')
    await readItem.actions.goToPage(0)

    readItem.close()
    await readItem.actions.focusSpans({ start: 40_512, end: 40_543 })

    expect(spans).toStrictEqual([])
  })
})

describe('a document the documents service fails to open', () => {
  const openWithAnswer = async (answer: typeof fetch): Promise<string> => {
    vi.useFakeTimers({ toFake: ['setTimeout'] })
    try {
      const api = createApi('md.numen.readers.pdf', {
        origin: 'http://core.invalid',
        fetch: answer,
      })
      const readItem = useDocumentReader(createDocuments(api), 'Gone.pdf')
      readItem.view.widen(800)
      const going = readItem.actions.goToPage(1)
      await vi.runAllTimersAsync()
      await going
      return readItem.state.error.value
    } finally {
      vi.useRealTimers()
    }
  }

  it('shows the generic text for an internal error from the backend', async () => {
    const message = await openWithAnswer(
      async () =>
        new Response(
          JSON.stringify({
            code: 'internal',
            message: 'pdfium: FPDF_LoadDocument /Users/someone/Vault/Gone.pdf: error 3',
          }),
          { status: 500, headers: { 'Content-Type': 'application/json' } },
        ),
    )

    expect(message).toContain('Numen is not responding')
    expect(message).not.toContain('/Users/someone')
  })

  it('shows the generic text for a fetch that fails before it reaches the core', async () => {
    const message = await openWithAnswer(async () => {
      throw new TypeError('Load failed')
    })

    expect(message).toContain('Numen is not responding')
    expect(message).not.toContain('Load failed')
  })
})
