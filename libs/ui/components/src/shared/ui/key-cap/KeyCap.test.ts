/**
 * What a cap says to a reader who is listening, and what it draws for one who
 * is looking.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import KeyCap from './KeyCap.vue'
import { ICONS } from './icons'
import type { Keystroke } from './keys'

const mountCap = (keys: Keystroke) => mount(KeyCap, { props: { keys } })

describe('what a reader who is listening receives', () => {
  it('is the keys in the order they are pressed, and then the letter', () => {
    const cap = mountCap({ icons: ['command', 'shift'], letter: 'P' })
    expect(cap.get('.sr-only').text()).toBe('Command Shift P')
  })

  it('is the keys alone where the stroke has no letter', () => {
    const cap = mountCap({ icons: ['return'], letter: '' })
    expect(cap.get('.sr-only').text()).toBe('Return')
  })

  it('reads out only the letter where no key is pressed', () => {
    const cap = mountCap({ icons: [], letter: 'K' })
    expect(cap.get('.sr-only').text()).toBe('K')
  })

  it('is nothing at all for an empty cap', () => {
    const cap = mountCap({ icons: [], letter: '' })
    expect(cap.get('.sr-only').text()).toBe('')
  })
})

describe('what is drawn', () => {
  it('draws an icon for each key pressed and hides it from a screen reader', () => {
    const cap = mountCap({ icons: ['control', 'option'], letter: 'G' })
    const icons = cap.findAll('.cap__icon')
    expect(icons).toHaveLength(2)
    for (const icon of icons) {
      expect(icon.attributes('aria-hidden')).toBe('true')
      expect(icon.attributes('focusable')).toBe('false')
    }
  })

  it('scales an icon by how much of the box it fills', () => {
    const cap = mountCap({ icons: ['control', 'shift'], letter: '' })
    const [control, shift] = cap.findAll('.cap__icon')
    expect(control?.attributes('style')).toContain(`--fills: ${ICONS.control.fills}`)
    expect(shift?.attributes('style')).toContain(`--fills: ${ICONS.shift.fills}`)
  })

  it('strokes an icon at the width its table gives it', () => {
    const cap = mountCap({ icons: ['control'], letter: '' })
    expect(cap.get('.cap__icon').attributes('stroke-width')).toBe(String(ICONS.control.stroke))
  })

  it('draws the letter after the icons, and none where the stroke has no letter', () => {
    const withLetter = mountCap({ icons: ['command'], letter: 'G' })
    expect(withLetter.get('span[aria-hidden="true"]').text()).toBe('G')

    const withoutLetter = mountCap({ icons: ['command'], letter: '' })
    expect(withoutLetter.find('span[aria-hidden="true"]').exists()).toBe(false)
  })

  it('is one keyboard element', () => {
    expect(mountCap({ icons: [], letter: 'A' }).element.tagName).toBe('KBD')
  })
})
