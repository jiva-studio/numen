/**
 * Which scrolls are initiated by the user.
 */
import { describe, expect, it } from 'vitest'
import { createScrollGate } from './scroll'

describe('the scroll gate', () => {
  it('does not take a scroll before any input as the person’s', () => {
    const gate = createScrollGate()
    gate.reset()
    expect(gate.isUserScroll()).toBe(false)
  })

  it('takes a scroll after input as the person’s', () => {
    const gate = createScrollGate()
    gate.reset()
    gate.noteInput()
    expect(gate.isUserScroll()).toBe(true)
  })

  it('forgets the input of an earlier opening when reset', () => {
    const gate = createScrollGate()
    gate.reset()
    gate.noteInput()
    gate.reset()
    expect(gate.isUserScroll()).toBe(false)
  })
})
