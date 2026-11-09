import { describe, expect, it } from 'vitest'
import { isKeyEvent, parseKey } from './parseKey'

const press = (init: KeyboardEventInit) => new KeyboardEvent('keydown', init)

describe('parseKey', () => {
  it('reads modifiers and a letter as the physical key of the letter', () => {
    expect(parseKey('mod+shift+p')).toStrictEqual({
      mod: true,
      ctrl: false,
      meta: false,
      alt: false,
      shift: true,
      code: 'KeyP',
    })
  })

  it('reads mod as its own modifier, apart from ctrl and meta', () => {
    expect(parseKey('mod+n')).toMatchObject({ mod: true, ctrl: false, meta: false })
  })

  it('reads ctrl, meta and alt as they are', () => {
    expect(parseKey('ctrl+alt+meta+x')).toMatchObject({
      mod: false,
      ctrl: true,
      alt: true,
      meta: true,
    })
  })

  it('reads a letter alone as a key without modifiers', () => {
    expect(parseKey('n')).toStrictEqual({
      mod: false,
      ctrl: false,
      meta: false,
      alt: false,
      shift: false,
      code: 'KeyN',
    })
  })

  it.each([
    ['5', 'Digit5'],
    ['f3', 'F3'],
    ['f12', 'F12'],
    ['enter', 'Enter'],
    ['space', 'Space'],
    ['up', 'ArrowUp'],
    ['pagedown', 'PageDown'],
    ['bracketleft', 'BracketLeft'],
    ['backquote', 'Backquote'],
  ])('reads the key name %s as %s', (name, code) => {
    expect(parseKey(`mod+${name}`)?.code).toBe(code)
  })

  it('ignores case and spaces around the parts', () => {
    expect(parseKey('Mod + Shift + P')).toMatchObject({ mod: true, shift: true, code: 'KeyP' })
  })

  it.each(['', 'mod+', 'mod+pp', 'f13', 'hyper+p', 'mod+shift', 'p+mod'])(
    'reads %j as nothing',
    (text) => {
      expect(parseKey(text)).toBeUndefined()
    },
  )
})

describe('isKeyEvent', () => {
  const chord = parseKey('mod+p')!

  it('is true for the code and the same modifiers', () => {
    expect(isKeyEvent(chord, press({ code: 'KeyP', ctrlKey: true }), false)).toBe(true)
  })

  it('is true on the Russian layout, where the key of the code KeyP types a different letter', () => {
    expect(isKeyEvent(chord, press({ code: 'KeyP', key: 'з', ctrlKey: true }), false)).toBe(true)
  })

  it('is false for the letter on another code, though the letter is the same', () => {
    expect(isKeyEvent(chord, press({ code: 'KeyZ', key: 'p', ctrlKey: true }), false)).toBe(false)
  })

  it('reads the letter of an event that has no code', () => {
    expect(isKeyEvent(chord, press({ key: 'P', ctrlKey: true }), false)).toBe(true)
    expect(isKeyEvent(chord, press({ key: 'з', ctrlKey: true }), false)).toBe(false)
  })

  it('is false with a missing or extra modifier', () => {
    expect(isKeyEvent(chord, press({ code: 'KeyP' }), false)).toBe(false)
    expect(isKeyEvent(chord, press({ code: 'KeyP', ctrlKey: true, shiftKey: true }), false)).toBe(false)
    expect(isKeyEvent(chord, press({ code: 'KeyP', ctrlKey: true, altKey: true }), false)).toBe(false)
    expect(isKeyEvent(chord, press({ code: 'KeyP', ctrlKey: true, metaKey: true }), false)).toBe(false)
  })

  it('matches mod to Meta on macOS and not to Ctrl', () => {
    expect(isKeyEvent(chord, press({ code: 'KeyP', metaKey: true }), true)).toBe(true)
    expect(isKeyEvent(chord, press({ code: 'KeyP', ctrlKey: true }), true)).toBe(false)
  })
})
