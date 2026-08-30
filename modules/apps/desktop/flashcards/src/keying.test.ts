import { describe, expect, it } from 'vitest'

import { asks, letterOf, picks, swallows } from './keying'

/** pressed is one keystroke, as the window meets it. */
const pressed = (key: string, more: Partial<KeyboardEvent> = {}) =>
  ({ key, repeat: false, altKey: false, ctrlKey: false, metaKey: false, ...more }) as KeyboardEvent

/** into is a keystroke that landed in something being written in. */
const into = (tag: string, written = false): Partial<KeyboardEvent> => ({
  target: { tagName: tag, isContentEditable: written } as unknown as EventTarget,
})

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

  it('brings the panel in on either side of the card', () => {
    expect(asks(pressed('a'), { shown: true })).toEqual({ does: 'ask' })
    expect(asks(pressed('A'), { shown: true })).toEqual({ does: 'ask' })
    expect(asks(pressed('a'), { shown: false })).toEqual({ does: 'ask' })
  })

  // With the panel up, escape sends it away and the sitting stays where it is.
  it('sends the panel away on escape before it leaves the sitting', () => {
    expect(asks(pressed('Escape'), { shown: true, asking: true })).toEqual({ does: 'shut' })
    expect(asks(pressed('Escape'), { shown: true, asking: false })).toEqual({ does: 'leave' })
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

  // A question is written in the panel with the same letters a card is
  // answered by, and there they are the question.
  it('asks for nothing while a question is being written', () => {
    for (const tag of ['INPUT', 'TEXTAREA']) {
      for (const key of ['1', '2', '3', '4', 'u', 'a', ' ']) {
        expect(asks(pressed(key, into(tag)), { shown: true })).toBeNull()
      }
    }
    expect(asks(pressed('1', into('DIV', true)), { shown: true })).toBeNull()
  })

  it('sends the panel away on escape from inside the field', () => {
    expect(asks(pressed('Escape', into('TEXTAREA')), { shown: true })).toEqual({ does: 'shut' })
  })

  it('swallows only the key it turns the card over with', () => {
    expect(swallows(asks(pressed(' '), { shown: false }))).toBe(true)
    expect(swallows(asks(pressed('3'), { shown: true }))).toBe(false)
    expect(swallows(null)).toBe(false)
  })
})

describe('the keys a deck is chosen with', () => {
  // The whole vault is the daily act, so it is the key under the hand.
  it('sits down to the whole vault on enter', () => {
    expect(picks(pressed('Enter'), 5)).toEqual({ does: 'all' })
    expect(picks(pressed(' '), 5)).toEqual({ does: 'all' })
  })

  // A person reads down the list and presses what they see.
  it('sits down to the deck a letter stands at', () => {
    expect(picks(pressed('a'), 5)).toEqual({ does: 'deck', at: 0 })
    expect(picks(pressed('c'), 5)).toEqual({ does: 'deck', at: 2 })
    expect(picks(pressed('e'), 5)).toEqual({ does: 'deck', at: 4 })
  })

  it('reads a letter typed in either case', () => {
    expect(picks(pressed('B'), 5)).toEqual({ does: 'deck', at: 1 })
  })

  it('picks no deck where the list holds none at that letter', () => {
    expect(picks(pressed('f'), 5)).toBeNull()
    expect(picks(pressed('z'), 5)).toBeNull()
    expect(picks(pressed('a'), 0)).toBeNull()
  })

  it('goes back to the vaults on escape', () => {
    expect(picks(pressed('Escape'), 5)).toEqual({ does: 'back' })
  })

  it('is not a choice from a key held down or pressed with a modifier', () => {
    expect(picks(pressed('a', { repeat: true }), 5)).toBeNull()
    expect(picks(pressed('a', { ctrlKey: true }), 5)).toBeNull()
    expect(picks(pressed('Enter', { metaKey: true }), 5)).toBeNull()
  })

  it('names each deck by the letter it is picked with', () => {
    expect(letterOf(0)).toBe('a')
    expect(letterOf(25)).toBe('z')
    // Past the alphabet a deck is picked with the hand.
    expect(letterOf(26)).toBe('')
  })
})
