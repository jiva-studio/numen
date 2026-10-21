/**
 * The book tabs of a window: what one is called, what it does when it comes on
 * screen, where a search sends the person, and what a command over one is asked
 * over.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useBookTab, type BookHandle } from './useBookTab'
import type { BookReaderState } from './useBookReader'

const readItem = (path: string, title = '', close = vi.fn()) =>
  ({ path, title: ref(title), close }) as unknown as BookReaderState

describe('what a book tab holds', () => {
  it('lays the columns out again once there is room to lay them out in', () => {
    const rendered: BookHandle = { measure: vi.fn(), handleKey: vi.fn(() => false) }
    const tab = useBookTab(readItem('library/Mahabharata.epub'))

    tab.measure()
    expect(rendered.measure).not.toHaveBeenCalled()

    tab.setBookHandle(rendered)
    tab.measure()
    expect(rendered.measure).toHaveBeenCalledTimes(1)
  })

  it('remains at the path the book is filed at', () => {
    expect(useBookTab(readItem('library/Mahabharata.epub')).path).toBe('library/Mahabharata.epub')
  })

  it('delegates focus and keypresses to the retained elements', () => {
    const focus = vi.fn()
    const element = { focus } as unknown as HTMLElement
    const handleKey = vi.fn(() => true)
    const rendered: BookHandle = { measure: vi.fn(), handleKey }
    const tab = useBookTab(readItem('library/Mahabharata.epub'))

    tab.setTabElement(element)
    tab.focusTab()
    expect(focus).toHaveBeenCalledTimes(1)

    const event = new KeyboardEvent('keydown', { key: 'ArrowRight' })
    expect(tab.handleKeyPress(event)).toBe(false)

    tab.setBookHandle(rendered)
    expect(tab.handleKeyPress(event)).toBe(true)
    expect(handleKey).toHaveBeenCalledWith(event)
  })
})

describe('a book tab with no element to focus', () => {
  it('focuses nothing and does not fail', () => {
    const tab = useBookTab(readItem('library/Mahabharata.epub'))

    expect(() => tab.focusTab()).not.toThrow()
  })
})
