/**
 * The book tabs of a window: what one is called, what it does when it comes on
 * screen, where a search sends the person, and what a command over one is asked
 * over.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'
import { useBookTab, type BookHandle, type BookTabState } from './useBookTab'
import { bookKind } from '../kind'
import { BOOK } from '@/entities/tab'
import type { BookReaderState } from './useBookReader'
import type { FileOpeners, SourceReader } from '@/entities/tab'
import type { Span } from '@/shared/span'
import type { WindowHandle } from '@/entities/tab'

const read = (path: string, title = '', close = vi.fn()) =>
  ({ path, title: ref(title), close }) as unknown as BookReaderState

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

const openers = () => {
  let reader: SourceReader | null = null
  const tabOpeners = {
    registerReader: (key: { format?: string }, opens: SourceReader) => {
      if (!key.format) return
      reader = opens
    },
  } as unknown as FileOpeners
  return { tabOpeners, opens: () => reader }
}

const settle = () => new Promise((done) => setTimeout(done, 0))

const createBookTabAt = (path: string, offsetVal: number, page: number, pages: number, length = 5_120_000) =>
  ({
    path,
    title: ref(''),
    offset: ref(offsetVal),
    span: computed(() => ({ begins: 0, ends: length })),
    page: computed(() => page),
    pages: computed(() => pages),
  }) as unknown as BookTabState

const kindOver = (held: BookTabState) => bookKind(createMockWindow(held).handle, () => held, openers().tabOpeners).kind

describe('what a book tab holds', () => {
  it('lays the columns out again once there is room to lay them out in', () => {
    const drawn: BookHandle = { measure: vi.fn(), handleKey: vi.fn(() => false) }
    const held = useBookTab(read('library/Mahabharata.epub'))

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
    const handleKey = vi.fn(() => true)
    const drawn: BookHandle = { measure: vi.fn(), handleKey }
    const held = useBookTab(read('library/Mahabharata.epub'))

    held.setTabElement(element)
    held.focusTab()
    expect(focus).toHaveBeenCalledTimes(1)

    const event = new KeyboardEvent('keydown', { key: 'ArrowRight' })
    expect(held.handleKeyPress(event)).toBe(false)

    held.setBookHandle(drawn)
    expect(held.handleKeyPress(event)).toBe(true)
    expect(handleKey).toHaveBeenCalledWith(event)
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
    const held = createBookTabAt('library/Mahabharata.epub', 1_200_000, 1_201, 5_000)

    expect(kindOver(held).attends?.(held)).toStrictEqual({
      path: 'library/Mahabharata.epub',
      book: { offset: 1_200_000, page: 1_201, pageCount: 5_000 },
    })
  })
})

describe('what a command over a book tab is asked over', () => {
  it('is the file the book stands in, as a source', () => {
    const held = createBookTabAt('library/Mahabharata.epub', 0, 1, 5_000)

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
    const focusSpans = vi.fn()
    const held = { focusSpans } as unknown as BookTabState
    const { handle, opened } = createMockWindow(held)
    const { tabOpeners, opens } = openers()
    bookKind(handle, () => held, tabOpeners)

    const spans: readonly Span[] = [{ from: 3_600, to: 3_642 }]
    opens()?.('library/Mahabharata.epub', spans)
    await settle()

    expect(opened).toStrictEqual([`${BOOK} library/Mahabharata.epub`])
    expect(focusSpans).toHaveBeenCalledWith({ from: 3_600, to: 3_642 })
  })
})
