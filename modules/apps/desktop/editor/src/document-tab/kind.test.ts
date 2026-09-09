/**
 * The document tabs of a window: what one is called, what it does when it comes
 * on screen, where a search sends the person, and what a command over one is
 * asked over.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { documenting, documentKind, type DocumentTabState, type PageHandle } from './kind'
import { DOCUMENT } from '../shared/tabs/workspace'
import type { OpenDocumentState } from './open'
import type { FileOpeners, SourceReader } from '../shared/tabs/openers'
import type { Span } from '../shared/core'
import type { WindowHandle } from '../shared/tabs/windowTabs'

/** A document being read, with only the parts a tab of it reaches for. */
const read = (path: string, close = vi.fn()) => ({ path, close }) as unknown as OpenDocumentState

/** A window, writing down what it was asked to open and holding what it made. */
const window_ = (held: DocumentTabState | null = null) => {
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

/** What puts documents in front, keeping the reader it is handed. */
const openers = () => {
  let reader: SourceReader | null = null
  const puts = { reads: (opens: SourceReader) => void (reader = opens) } as unknown as FileOpeners
  return { puts, opens: () => reader }
}

/** A moment for whatever the reader asked the window for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** A document open at a page of a file, as far as the window reads one. */
const openedAt = (path: string, page: number, pageCount: number) =>
  ({ path, at: ref(page), pages: ref(Array.from({ length: pageCount })) }) as unknown as DocumentTabState

/** The kind, over a window holding the document it is handed. */
const kindOver = (held: DocumentTabState) => documentKind(window_(held).handle, () => held, openers().puts).kind

describe('what a document tab holds', () => {
  it('measures the page again once there is a page to measure', () => {
    const page: PageHandle = { measure: vi.fn() }
    const held = documenting(read('physics/Boltzmann.pdf'))

    // Drawn nowhere yet, and asked to measure all the same.
    held.measure()
    expect(page.measure).not.toHaveBeenCalled()

    held.drew(page)
    held.measure()
    expect(page.measure).toHaveBeenCalledTimes(1)
  })

  // A tab is drawn while it is out of sight, where there is no room to lay a
  // page out in, so what it drew is let go of when it goes.
  it('measures nothing once the page it drew is gone', () => {
    const page: PageHandle = { measure: vi.fn() }
    const held = documenting(read('physics/Boltzmann.pdf'))
    held.drew(page)
    held.drew(null)

    held.measure()
    expect(page.measure).not.toHaveBeenCalled()
  })

  it('is the document it was opened on', () => {
    expect(documenting(read('physics/Boltzmann.pdf')).path).toBe('physics/Boltzmann.pdf')
  })
})

describe('a document tab', () => {
  const kindOf = () => {
    const { handle } = window_()
    const { puts } = openers()
    return documentKind(handle, (path) => documenting(read(path)), puts)
  }

  it('is called by the file and not by the folders above it', () => {
    const { kind } = kindOf()
    expect(kind.called(documenting(read('physics/heat/Boltzmann 1877.pdf')))).toBe(
      'Boltzmann 1877.pdf',
    )
    expect(kind.called(documenting(read('Boltzmann.pdf')))).toBe('Boltzmann.pdf')
  })

  // A document is its own tab, so the same one opened again is the tab it is
  // already read in.
  it('is one tab per document', () => {
    const { kind } = kindOf()
    expect(kind.kind).toBe(DOCUMENT)
    expect(kind.identity?.('physics/Boltzmann.pdf')).toBe('physics/Boltzmann.pdf')
  })

  it('measures the page when the tab comes on screen', () => {
    const { kind } = kindOf()
    const page: PageHandle = { measure: vi.fn() }
    const held = documenting(read('physics/Boltzmann.pdf'))
    held.drew(page)

    kind.shown?.(held, 'a tab')
    expect(page.measure).toHaveBeenCalledTimes(1)
  })

  it('lets go of the document it was reading, and goes', () => {
    const { kind } = kindOf()
    const close = vi.fn()
    const held = documenting(read('physics/Boltzmann.pdf', close))

    expect(kind.shuts?.(held, 'a tab')).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('a search that landed in a document', () => {
  it('opens the document and turns it to what was found', async () => {
    const reached = vi.fn()
    const held = { reach: reached } as unknown as DocumentTabState
    const { handle, opened } = window_(held)
    const { puts, opens } = openers()
    documentKind(handle, (path) => documenting(read(path)), puts)

    const spans: readonly Span[] = [
      { from: 0, to: 12 },
      { from: 400, to: 420 },
    ]
    opens()?.('physics/Boltzmann.pdf', spans)
    await settles()

    expect(opened).toEqual(['document physics/Boltzmann.pdf'])
    expect(reached).toHaveBeenCalledWith(...spans)
  })
})

describe('what a command asked over a document tab is over', () => {
  it('is the file it reads, which is what a run is asked over', () => {
    const held = openedAt('Ants.epub', 3, 40)

    expect(kindOver(held).over!(held)).toStrictEqual({ file: 'Ants.epub', source: 'book' })
  })
})

describe('what a document tab holds, as whoever answers for the person is told it', () => {
  it('is the file, the page in front of them, and how many pages there are', () => {
    const held = openedAt('Ants.epub', 3, 40)

    expect(kindOver(held).attends!(held)).toStrictEqual({
      path: 'Ants.epub',
      document: { page: 4, pageCount: 40 },
    })
  })
})
