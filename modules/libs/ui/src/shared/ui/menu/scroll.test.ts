/**
 * Which scrolls are the person's.
 */
import { describe, expect, it } from 'vitest'
import { createScrollGate } from './scroll'

describe('the scroll gate', () => {
  it('does not take a scroll before any input as the person’s', () => {
    const gate = createScrollGate()
    gate.arm()
    expect(gate.isUserScroll()).toBe(false)
  })

  it('takes a scroll after input as the person’s', () => {
    const gate = createScrollGate()
    gate.arm()
    gate.noteInput()
    expect(gate.isUserScroll()).toBe(true)
  })

  it('forgets the input of an earlier opening when armed again', () => {
    const gate = createScrollGate()
    gate.arm()
    gate.noteInput()
    gate.arm()
    expect(gate.isUserScroll()).toBe(false)
  })
})
