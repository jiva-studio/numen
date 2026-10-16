/**
 * The book tabs of a window: what one is called, what it does when it comes on
 * screen, where a search sends the person, and what a command over one is asked
 * over.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useBookTab, type BookHandle } from './useBookTab'
import type { BookReaderState } from './useBookReader'

const read = (path: string, title = '', close = vi.fn()) =>
  ({ path, title: ref(title), close }) as unknown as BookReaderState

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

describe('a book tab with no element to focus', () => {
  it('focuses nothing and does not fail', () => {
    const held = useBookTab(read('library/Mahabharata.epub'))

    expect(() => held.focusTab()).not.toThrow()
  })
})
