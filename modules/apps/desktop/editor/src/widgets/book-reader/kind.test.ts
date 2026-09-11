/**
 * The book tabs of a window: what one is called, what it does when it comes on
 * screen, where a search sends the person, and what a command over one is asked
 * over.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'
import { bookKind, useBookTab, type BookHandle, type BookTabState } from './kind'
import { BOOK } from '../../shared/tabs/workspace'
import type { BookReaderState } from './open'
import type { FileOpeners, SourceReader } from '../../shared/tabs/openers'
import type { Span } from '../../shared/core'
import type { WindowHandle } from '../../shared/tabs/windowTabs'

/** A book being read, with only the parts a tab of it reaches for. */
const read = (path: string, title = '', close = vi.fn()) =>
  ({ path, title: ref(title), close }) as unknown as BookReaderState

/** A mock window handle writing down opened tabs and returning a given tab state. */
const createMockWindow = (held?: BookTabState) => {
  const opened: string[] = []
  const handle = {
    opens: async (kind: string, at?: string) => {
      opened.push(`${kind} ${at ?? ''}`.trim())
      return `id of ${at}`
    },
    holds: () => held ?? null,
  } as unknown as WindowHandle
  return { handle, opened }
}

/** What puts books in front, keeping the reader it is handed. */
const openers = () => {
  let reader: SourceReader | null = null
  const puts = {
    reads: (key: { format?: string }, opens: SourceReader) => {
      if (!key.format) return
      reader = opens
    },
  } as unknown as FileOpeners
  return { puts, opens: () => reader }
}

/** A moment for whatever the reader asked the window for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** A book open at an offset of a file, as far as the window reads one. */
const openedAt = (path: string, at: number, page: number, pages: number, length = 5_120_000) =>
  ({
    path,
    title: ref(''),
    at: ref(at),
    span: computed(() => ({ begins: 0, ends: length })),
    page: computed(() => page),
    pages: computed(() => pages),
  }) as unknown as BookTabState

/** The kind, over a window holding the book it is handed. */
const kindOver = (held: BookTabState) => bookKind(createMockWindow(held).handle, () => held, openers().puts).kind

describe('what a book tab holds', () => {
  it('lays the columns out again once there is room to lay them out in', () => {
    const drawn: BookHandle = { measure: vi.fn(), pressed: vi.fn(() => false) }
    const held = useBookTab(read('library/Mahabharata.epub'))

    // Drawn nowhere yet, and asked to measure all the same.
    held.measure()
    expect(drawn.measure).not.toHaveBeenCalled()

    held.setBookHandle(drawn)
    held.measure()
    expect(drawn.measure).toHaveBeenCalledTimes(1)
  })

  it('stands at the path the book is filed at', () => {
    expect(useBookTab(read('library/Mahabharata.epub')).path).toBe('library/Mahabharata.epub')
  })

  it('delegates focus and keypresses to the held elements', () => {
    const focus = vi.fn()
    const element = { focus } as unknown as HTMLElement
    const pressed = vi.fn(() => true)
    const drawn: BookHandle = { measure: vi.fn(), pressed }
    const held = useBookTab(read('library/Mahabharata.epub'))

    held.setTabElement(element)
    held.focusTab()
    expect(focus).toHaveBeenCalledTimes(1)

    const event = new KeyboardEvent('keydown', { key: 'ArrowRight' })
    expect(held.handleKeyPress(event)).toBe(false)

    held.setBookHandle(drawn)
    expect(held.handleKeyPress(event)).toBe(true)
    expect(pressed).toHaveBeenCalledWith(event)
  })
})

describe('what a book tab is called', () => {
  it('is what the book calls itself', () => {
    const held = useBookTab(read('library/mbh-04.epub', 'Virāṭa Parva'))

    expect(kindOver(held).called?.(held)).toBe('Virāṭa Parva')
  })

  it('is the name of the file, for a book that calls itself nothing', () => {
    const held = useBookTab(read('library/sub/mbh-04.epub'))

    expect(kindOver(held).called?.(held)).toBe('mbh-04.epub')
  })
})

describe('what a book tab tells whoever answers for the person', () => {
  it('is the offset in front, and the page it falls on', () => {
    const held = openedAt('library/Mahabharata.epub', 1_200_000, 1_201, 5_000)

    expect(kindOver(held).attends?.(held)).toStrictEqual({
      path: 'library/Mahabharata.epub',
      book: { offset: 1_200_000, page: 1_201, pageCount: 5_000 },
    })
  })
})

describe('what a command over a book tab is asked over', () => {
  it('is the file the book stands in, as a source', () => {
    const held = openedAt('library/Mahabharata.epub', 0, 1, 5_000)

    expect(kindOver(held).over?.(held)).toStrictEqual({
      file: 'library/Mahabharata.epub',
      source: 'book',
    })
  })
})

describe('a book let go of', () => {
  it('lets go of what it held and leaves the pane', () => {
    const close = vi.fn()
    const held = useBookTab(read('library/Mahabharata.epub', '', close))

    expect(kindOver(held).shuts?.(held, 'a tab')).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('a passage of a book reached', () => {
  it('opens the book and sends the tab to the spans asked about', async () => {
    const reach = vi.fn()
    const held = { reach } as unknown as BookTabState
    const { handle, opened } = createMockWindow(held)
    const { puts, opens } = openers()
    bookKind(handle, () => held, puts)

    const spans: readonly Span[] = [{ from: 3_600, to: 3_642 }]
    opens()?.('library/Mahabharata.epub', spans)
    await settles()

    expect(opened).toStrictEqual([`${BOOK} library/Mahabharata.epub`])
    expect(reach).toHaveBeenCalledWith({ from: 3_600, to: 3_642 })
  })
})
