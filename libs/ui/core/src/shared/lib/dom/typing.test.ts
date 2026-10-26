import { describe, expect, it } from 'vitest'
import { isTyping } from './typing'

/** A keystroke whose target is the element named. */
const createPress = (target: unknown) => ({ target }) as unknown as KeyboardEvent

describe('a key pressed into something being written in', () => {
  it('is a key typed into a field, an area of text or an editable element', () => {
    expect(isTyping(createPress({ tagName: 'INPUT' }))).toBe(true)
    expect(isTyping(createPress({ tagName: 'TEXTAREA' }))).toBe(true)
    expect(isTyping(createPress({ tagName: 'DIV', isContentEditable: true }))).toBe(true)
  })

  it('is not a key pressed on anything else, or on nothing', () => {
    expect(isTyping(createPress({ tagName: 'BUTTON', isContentEditable: false }))).toBe(false)
    expect(isTyping(createPress(null))).toBe(false)
  })
})
