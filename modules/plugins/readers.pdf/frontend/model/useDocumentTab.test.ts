/**
 * The document tabs of a window.
 */
import { describe, expect, it, vi } from 'vitest'
import { useDocumentTab, type PageHandle } from './useDocumentTab'
import type { DocumentReaderState } from './useDocumentReader'

const read = (path: string, close = vi.fn()) => ({ path, close }) as unknown as DocumentReaderState

/** The pages drawn in a tab, which record what was asked of them. */
const createPages = (): PageHandle => ({
  measure: vi.fn(),
  handleKey: vi.fn(() => false),
  focusPages: vi.fn(),
})

describe('what a document tab holds', () => {
  it('lays the page out again once there is room to lay it out in', () => {
    const pages = createPages()
    const held = useDocumentTab(read('doc.pdf'))

    held.measure()
    expect(pages.measure).not.toHaveBeenCalled()

    held.setPageHandle(pages)
    held.measure()
    expect(pages.measure).toHaveBeenCalledTimes(1)
  })

  it('remains at the path the document is filed at', () => {
    expect(useDocumentTab(read('doc.pdf')).path).toBe('doc.pdf')
  })

  it('delegates keypresses to the page', () => {
    const handleKey = vi.fn(() => true)
    const pages: PageHandle = { ...createPages(), handleKey }
    const held = useDocumentTab(read('doc.pdf'))

    const event = new KeyboardEvent('keydown', { key: 'ArrowRight' })
    expect(held.handleKeyPress(event)).toBe(false)

    held.setPageHandle(pages)
    expect(held.handleKeyPress(event)).toBe(true)
    expect(handleKey).toHaveBeenCalledWith(event)
  })

  it('delegates focus to the page', () => {
    const focusPages = vi.fn()
    const pages: PageHandle = { ...createPages(), focusPages }
    const held = useDocumentTab(read('doc.pdf'))

    held.focusTab()
    expect(focusPages).not.toHaveBeenCalled()

    held.setPageHandle(pages)
    held.focusTab()
    expect(focusPages).toHaveBeenCalledTimes(1)
  })
})
