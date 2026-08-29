import { describe, expect, it } from 'vitest'

import { asks, swallows } from './keying'

/** pressed is one keystroke, as the window meets it. */
const pressed = (key: string, more: Partial<KeyboardEvent> = {}) =>
  ({ key, repeat: false, altKey: false, ctrlKey: false, metaKey: false, ...more }) as KeyboardEvent

describe('the keys a sitting is done with', () => {
  it('turns the card over with the space bar, and only while it is face up', () => {
    expect(asks(pressed(' '), { shown: false })).toEqual({ does: 'show' })
    expect(asks(pressed(' '), { shown: true })).toBeNull()
  })

  it('says how the card went, by the number of the answer', () => {
    expect(asks(pressed('1'), { shown: true })).toEqual({ does: 'answer', how: 'again' })
    expect(asks(pressed('2'), { shown: true })).toEqual({ does: 'answer', how: 'hard' })
    expect(asks(pressed('3'), { shown: true })).toEqual({ does: 'answer', how: 'good' })
    expect(asks(pressed('4'), { shown: true })).toEqual({ does: 'answer', how: 'easy' })
  })

  it('is not answered by a number that names none of the four', () => {
    expect(asks(pressed('0'), { shown: true })).toBeNull()
    expect(asks(pressed('5'), { shown: true })).toBeNull()
    expect(asks(pressed('9'), { shown: true })).toBeNull()
  })

  it('takes the last answer back, in either case of the letter', () => {
    expect(asks(pressed('u'), { shown: true })).toEqual({ does: 'takeBack' })
    expect(asks(pressed('U'), { shown: true })).toEqual({ does: 'takeBack' })
  })

  it('leaves the sitting on escape', () => {
    expect(asks(pressed('Escape'), { shown: true })).toEqual({ does: 'leave' })
  })

  // A key held down repeats, and a card is answered once.
  it('is not answered again by a key held down', () => {
    expect(asks(pressed('3', { repeat: true }), { shown: true })).toBeNull()
    expect(asks(pressed(' ', { repeat: true }), { shown: false })).toBeNull()
  })

  // A key pressed with a modifier is the machine's own shortcut.
  it('is not answered by a key pressed with a modifier', () => {
    for (const held of ['altKey', 'ctrlKey', 'metaKey'] as const) {
      expect(asks(pressed('3', { [held]: true }), { shown: true })).toBeNull()
      expect(asks(pressed('u', { [held]: true }), { shown: true })).toBeNull()
    }
  })

  // Once the card is over, enter belongs to whatever the person has moved
  // focus to: a button reached with the keyboard is pressed with the keyboard.
  it('leaves enter alone', () => {
    expect(asks(pressed('Enter'), { shown: false })).toBeNull()
    expect(asks(pressed('Enter'), { shown: true })).toBeNull()
  })

  it('swallows only the key it turns the card over with', () => {
    expect(swallows(asks(pressed(' '), { shown: false }))).toBe(true)
    expect(swallows(asks(pressed('3'), { shown: true }))).toBe(false)
    expect(swallows(null)).toBe(false)
  })
})
