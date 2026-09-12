/**
 * The document tabs of a window: what one is called, what it does when it comes
 * on screen, where a search sends the person, and what a command over one is
 * asked over.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useDocumentTab, type DocumentTabState, type PageHandle } from './useDocumentTab'
import { documentKind } from '../kind'
import { DOCUMENT } from '@/entities/tab'
import type { DocumentReaderState } from './useDocumentReader'
import type { FileOpeners, SourceReader } from '@/entities/tab'
import type { Span } from '@/shared/span'
import type { WindowHandle } from '@/entities/tab'

const read = (path: string, close = vi.fn()) => ({ path, close }) as unknown as DocumentReaderState

const createMockDocumentWindow = (held: DocumentTabState | null = null) => {
  const opened: string[] = []
  const handle = {
    opens: async (kind: string, at?: string) => {
      opened.push(`${kind} ${at ?? ''}`.trim())
      return `id of ${at}`
    },
    holds: () => held,
  } as unknown as WindowHandle
  return { handle, opened }
}

const openers = () => {
  let reader: SourceReader | null = null
  const tabOpeners = {
    registerReader: (key: { format?: string }, opens: SourceReader) => {
      if (key.format) return
      reader = opens
    },
  } as unknown as FileOpeners
  return { tabOpeners, opens: () => reader }
}

const settles = () => new Promise((done) => setTimeout(done, 0))

const createDocumentTabAt = (path: string, page: number, pageCount: number) =>
  ({ path, pageNumber: ref(page), pages: ref(Array.from({ length: pageCount })) }) as unknown as DocumentTabState

const kindOver = (held: DocumentTabState) => documentKind(createMockDocumentWindow(held).handle, () => held, openers().tabOpeners).kind

describe('what a document tab holds', () => {
  it('measures the page again once there is a page to measure', () => {
    const page: PageHandle = { measure: vi.fn() }
    const held = useDocumentTab(read('physics/Boltzmann.pdf'))

    held.measure()
    expect(page.measure).not.toHaveBeenCalled()

    held.setPageHandle(page)
    held.measure()
    expect(page.measure).toHaveBeenCalledTimes(1)
  })

  it('measures nothing once the page it drew is gone', () => {
    const page: PageHandle = { measure: vi.fn() }
    const held = useDocumentTab(read('physics/Boltzmann.pdf'))
    held.setPageHandle(page)
    held.setPageHandle(null)

    held.measure()
    expect(page.measure).not.toHaveBeenCalled()
  })

  it('is the document it was opened on', () => {
    expect(useDocumentTab(read('physics/Boltzmann.pdf')).path).toBe('physics/Boltzmann.pdf')
  })
})

describe('a document tab', () => {
  const kindOf = () => {
    const { handle } = createMockDocumentWindow()
    const { tabOpeners } = openers()
    return documentKind(handle, (path) => useDocumentTab(read(path)), tabOpeners)
  }

  it('is called by the file and not by the folders above it', () => {
    const { kind } = kindOf()
    expect(kind.called?.(useDocumentTab(read('physics/heat/Boltzmann 1877.pdf')))).toBe(
      'Boltzmann 1877.pdf',
    )
    expect(kind.called?.(useDocumentTab(read('Boltzmann.pdf')))).toBe('Boltzmann.pdf')
  })

  it('is one tab per document', () => {
    const { kind } = kindOf()
    expect(kind.kind).toBe(DOCUMENT)
    expect(kind.identity?.('physics/Boltzmann.pdf')).toBe('physics/Boltzmann.pdf')
  })

  it('measures the page when the tab comes on screen', () => {
    const { kind } = kindOf()
    const page: PageHandle = { measure: vi.fn() }
    const held = useDocumentTab(read('physics/Boltzmann.pdf'))
    held.setPageHandle(page)

    kind.shown?.(held, 'a tab')
    expect(page.measure).toHaveBeenCalledTimes(1)
  })

  it('lets go of the document it was reading, and goes', () => {
    const { kind } = kindOf()
    const close = vi.fn()
    const held = useDocumentTab(read('physics/Boltzmann.pdf', close))

    expect(kind.shuts?.(held, 'a tab')).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('a search that landed in a document', () => {
  it('opens the document and turns it to what was found', async () => {
    const focused = vi.fn()
    const held = { focusSpans: focused } as unknown as DocumentTabState
    const { handle, opened } = createMockDocumentWindow(held)
    const { tabOpeners, opens } = openers()
    documentKind(handle, (path) => useDocumentTab(read(path)), tabOpeners)

    const spans: readonly Span[] = [
      { from: 0, to: 12 },
      { from: 400, to: 420 },
    ]
    opens()?.('physics/Boltzmann.pdf', spans)
    await settles()

    expect(opened).toEqual(['document physics/Boltzmann.pdf'])
    expect(focused).toHaveBeenCalledWith(...spans)
  })
})

describe('what a command asked over a document tab is over', () => {
  it('is the file it reads, which is what a run is asked over', () => {
    const held = createDocumentTabAt('Ants.epub', 3, 40)

    expect(kindOver(held).over!(held)).toStrictEqual({ file: 'Ants.epub', source: 'book' })
  })
})

describe('what a document tab holds, as whoever answers for the person is told it', () => {
  it('is the file, the page in front of them, and how many pages there are', () => {
    const held = createDocumentTabAt('Ants.epub', 3, 40)

    expect(kindOver(held).attends!(held)).toStrictEqual({
      path: 'Ants.epub',
      document: { page: 4, pageCount: 40 },
    })
  })
})
