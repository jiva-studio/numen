import { describe, expect, it } from 'vitest'

import { getPickerKeyIntent, getSessionKeyIntent, isSwallowed, letterOf } from './keys'

/** One keystroke, as the window meets it. */
const createPress = (key: string, more: Partial<KeyboardEvent> = {}) =>
  ({
    key,
    repeat: false,
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    shiftKey: false,
    ...more,
  }) as KeyboardEvent

/** The target of a keystroke that landed in something being written in. */
const createTarget = (tag: string, isEditable = false): Partial<KeyboardEvent> => ({
  target: { tagName: tag, isContentEditable: isEditable } as unknown as EventTarget,
})

describe('the keys a session is done with', () => {
  it('turns the card over with the space bar, and only while it is face up', () => {
    expect(getSessionKeyIntent(createPress(' '), { isShown: false })).toEqual({ does: 'show' })
    expect(getSessionKeyIntent(createPress(' '), { isShown: true })).toBeNull()
  })

  it('says how the card went, by the number of the answer', () => {
    expect(getSessionKeyIntent(createPress('1'), { isShown: true })).toEqual({
      does: 'answer',
      how: 'again',
    })
    expect(getSessionKeyIntent(createPress('2'), { isShown: true })).toEqual({
      does: 'answer',
      how: 'hard',
    })
    expect(getSessionKeyIntent(createPress('3'), { isShown: true })).toEqual({
      does: 'answer',
      how: 'good',
    })
    expect(getSessionKeyIntent(createPress('4'), { isShown: true })).toEqual({
      does: 'answer',
      how: 'easy',
    })
  })

  it('is not answered by a number that names none of the four', () => {
    expect(getSessionKeyIntent(createPress('0'), { isShown: true })).toBeNull()
    expect(getSessionKeyIntent(createPress('5'), { isShown: true })).toBeNull()
    expect(getSessionKeyIntent(createPress('9'), { isShown: true })).toBeNull()
  })

  it('takes the last answer back, in either case of the letter', () => {
    expect(getSessionKeyIntent(createPress('u'), { isShown: true })).toEqual({ does: 'takeBack' })
    expect(getSessionKeyIntent(createPress('U'), { isShown: true })).toEqual({ does: 'takeBack' })
  })

  it('leaves the session on escape', () => {
    expect(getSessionKeyIntent(createPress('Escape'), { isShown: true })).toEqual({ does: 'leave' })
  })

  // The letters on their own are what a card is answered by, so the panels are
  // held with the overlay key: control here, command on a Mac.
  it('brings the panels in on either side of the card, held with the overlay key', () => {
    for (const held of ['ctrlKey', 'metaKey'] as const) {
      expect(getSessionKeyIntent(createPress('a', { [held]: true }), { isShown: true })).toEqual({
        does: 'ask',
      })
      expect(getSessionKeyIntent(createPress('A', { [held]: true }), { isShown: false })).toEqual({
        does: 'ask',
      })
      expect(getSessionKeyIntent(createPress('r', { [held]: true }), { isShown: true })).toEqual({
        does: 'read',
      })
      expect(getSessionKeyIntent(createPress('R', { [held]: true }), { isShown: false })).toEqual({
        does: 'read',
      })
    }
  })

  it('is not asked for by the letters on their own', () => {
    expect(getSessionKeyIntent(createPress('a'), { isShown: true })).toBeNull()
    expect(getSessionKeyIntent(createPress('r'), { isShown: true })).toBeNull()
  })

  // Control and A is how a person selects what they have written, and the field
  // is the one place in this window anything is written.
  it('leaves the overlay key to the field a question is written in', () => {
    for (const tag of ['INPUT', 'TEXTAREA']) {
      expect(
        getSessionKeyIntent(createPress('a', { ...createTarget(tag), ctrlKey: true }), {
          isShown: true,
        }),
      ).toBeNull()
      expect(
        getSessionKeyIntent(createPress('r', { ...createTarget(tag), metaKey: true }), {
          isShown: true,
        }),
      ).toBeNull()
    }
  })

  // With a panel up, escape sends it away and the session stays where it is.
  it('sends the panel away on escape before it leaves the session', () => {
    expect(getSessionKeyIntent(createPress('Escape'), { isShown: true, isAsking: true })).toEqual({
      does: 'shut',
    })
    expect(getSessionKeyIntent(createPress('Escape'), { isShown: true, isReading: true })).toEqual({
      does: 'shut',
    })
    expect(getSessionKeyIntent(createPress('Escape'), { isShown: true, isAsking: false })).toEqual({
      does: 'leave',
    })
  })

  // The reading is read down, and space is the key the hand is already on. The
  // answer is still shown by the control standing under both panes.
  it('scrolls the reading with the space bar instead of turning the card', () => {
    expect(getSessionKeyIntent(createPress(' '), { isShown: false, isReading: true })).toEqual({
      does: 'scroll',
      back: false,
    })
    expect(
      getSessionKeyIntent(createPress(' ', { shiftKey: true }), { isShown: true, isReading: true }),
    ).toEqual({
      does: 'scroll',
      back: true,
    })
  })

  // A key held down repeats, and a card is answered once.
  it('is not answered again by a key held down', () => {
    expect(getSessionKeyIntent(createPress('3', { repeat: true }), { isShown: true })).toBeNull()
    expect(getSessionKeyIntent(createPress(' ', { repeat: true }), { isShown: false })).toBeNull()
  })

  // A key pressed with a modifier is the machine's own shortcut.
  it('is not answered by a key pressed with a modifier', () => {
    for (const held of ['altKey', 'ctrlKey', 'metaKey'] as const) {
      expect(getSessionKeyIntent(createPress('3', { [held]: true }), { isShown: true })).toBeNull()
      expect(getSessionKeyIntent(createPress('u', { [held]: true }), { isShown: true })).toBeNull()
    }
  })

  // Once the card is over, enter belongs to whatever the person has moved
  // focus to: a button reached with the keyboard is pressed with the keyboard.
  it('leaves enter alone', () => {
    expect(getSessionKeyIntent(createPress('Enter'), { isShown: false })).toBeNull()
    expect(getSessionKeyIntent(createPress('Enter'), { isShown: true })).toBeNull()
  })

  // A question is written in the panel with the same letters a card is
  // answered by, and there they are the question.
  it('asks for nothing while a question is being written', () => {
    for (const tag of ['INPUT', 'TEXTAREA']) {
      for (const key of ['1', '2', '3', '4', 'u', 'a', ' ']) {
        expect(
          getSessionKeyIntent(createPress(key, createTarget(tag)), { isShown: true }),
        ).toBeNull()
      }
    }
    expect(
      getSessionKeyIntent(createPress('1', createTarget('DIV', true)), { isShown: true }),
    ).toBeNull()
  })

  it('sends the panel away on escape from inside the field', () => {
    expect(
      getSessionKeyIntent(createPress('Escape', createTarget('TEXTAREA')), { isShown: true }),
    ).toEqual({ does: 'shut' })
  })

  it('swallows only the keys the page would act on itself', () => {
    expect(isSwallowed(getSessionKeyIntent(createPress(' '), { isShown: false }))).toBe(true)
    // The page scrolls itself on space, and the reading is what is scrolled.
    expect(
      isSwallowed(getSessionKeyIntent(createPress(' '), { isShown: true, isReading: true })),
    ).toBe(true)
    expect(isSwallowed(getSessionKeyIntent(createPress('3'), { isShown: true }))).toBe(false)
    expect(isSwallowed(null)).toBe(false)
  })
})

describe('the keys a deck is chosen with', () => {
  // The whole vault is the daily act, so it is the key under the hand.
  it('sits down to the whole vault on enter', () => {
    expect(getPickerKeyIntent(createPress('Enter'), 5)).toEqual({ does: 'all' })
    expect(getPickerKeyIntent(createPress(' '), 5)).toEqual({ does: 'all' })
  })

  // A person reads down the list and presses what they see.
  it('sits down to the deck a letter stands at', () => {
    expect(getPickerKeyIntent(createPress('a'), 5)).toEqual({ does: 'deck', at: 0 })
    expect(getPickerKeyIntent(createPress('c'), 5)).toEqual({ does: 'deck', at: 2 })
    expect(getPickerKeyIntent(createPress('e'), 5)).toEqual({ does: 'deck', at: 4 })
  })

  it('reads a letter typed in either case', () => {
    expect(getPickerKeyIntent(createPress('B'), 5)).toEqual({ does: 'deck', at: 1 })
  })

  it('picks no deck where the list holds none at that letter', () => {
    expect(getPickerKeyIntent(createPress('f'), 5)).toBeNull()
    expect(getPickerKeyIntent(createPress('z'), 5)).toBeNull()
    expect(getPickerKeyIntent(createPress('a'), 0)).toBeNull()
  })

  it('goes back to the vaults on escape', () => {
    expect(getPickerKeyIntent(createPress('Escape'), 5)).toEqual({ does: 'back' })
  })

  it('is not a choice from a key held down or pressed with a modifier', () => {
    expect(getPickerKeyIntent(createPress('a', { repeat: true }), 5)).toBeNull()
    expect(getPickerKeyIntent(createPress('a', { ctrlKey: true }), 5)).toBeNull()
    expect(getPickerKeyIntent(createPress('Enter', { metaKey: true }), 5)).toBeNull()
  })

  it('names each deck by the letter it is picked with', () => {
    expect(letterOf(0)).toBe('a')
    expect(letterOf(25)).toBe('z')
    // Past the alphabet a deck is picked with the hand.
    expect(letterOf(26)).toBe('')
  })
})
