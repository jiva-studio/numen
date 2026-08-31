/**
 * The keystrokes that reach a command, asked without a window.
 *
 * The failure this is here for is a key drawn on a row that does nothing: the
 * table is read twice, once to bind and once to draw, and a letter naming a
 * command the window does not have would draw a cap over silence.
 */
import { describe, expect, it } from 'vitest'
import { chorded, commandFor, keyOf, keysOf, CHORDS } from './keying'
import { commandsOf } from './commanding'
import { WORDS as words } from './words'

const pressing = (over: Partial<KeyboardEventInit> = {}) => ({
  altKey: false,
  ctrlKey: false,
  metaKey: false,
  ...over,
})

describe('a keystroke the window answers', () => {
  it('is one letter held with Control, or with Command', () => {
    expect(chorded(pressing({ ctrlKey: true }))).toBe(true)
    expect(chorded(pressing({ metaKey: true }))).toBe(true)
  })

  it('belongs to nobody while Alt is held with it', () => {
    expect(chorded(pressing({ ctrlKey: true, altKey: true }))).toBe(false)
    expect(chorded(pressing({ metaKey: true, altKey: true }))).toBe(false)
  })

  it('belongs to nobody with neither of the two held', () => {
    expect(chorded(pressing())).toBe(false)
  })
})

describe('the command a letter asks for', () => {
  it('is the one the chord beside it names', () => {
    expect(commandFor('n', false)).toBe('note')
    expect(commandFor('g', false)).toBe('goto')
  })

  it('is the same command in either case', () => {
    expect(commandFor('N', false)).toBe('note')
  })

  it('is another command entirely when Shift is held with the letter', () => {
    expect(commandFor('p', true)).toBe('travel')
    expect(commandFor('c', true)).toBe('child')
    expect(commandFor('a', true)).toBe('agent')
    expect(commandFor('w', true)).toBe('close')
    expect(commandFor('n', true)).toBe('newVault')
  })

  it('is nothing when a letter is held with the wrong half of the chord', () => {
    expect(commandFor('g', true)).toBe('')
    expect(commandFor('c', false)).toBe('')
    expect(commandFor('w', false)).toBe('')
  })

  it('is nothing where no command answers to the letter', () => {
    expect(commandFor('q', false)).toBe('')
    expect(commandFor('q', true)).toBe('')
  })

  it('is nothing for the two letters the window keeps for itself', () => {
    expect(commandFor('k', false)).toBe('')
    expect(commandFor('p', false)).toBe('')
  })
})

describe('how a keystroke is drawn on the row that names it', () => {
  it('is the key the keyboard in hand holds, and the letter in capitals', () => {
    expect(keyOf('note', 'MacIntel')).toEqual({ marks: ['command'], letter: 'N' })
    expect(keyOf('note', 'Linux x86_64')).toEqual({ marks: ['control'], letter: 'N' })
    expect(keyOf('goto', 'Linux x86_64')).toEqual({ marks: ['control'], letter: 'G' })
  })

  it('carries Shift where the chord holds it', () => {
    expect(keyOf('travel', 'MacIntel')).toEqual({ marks: ['command', 'shift'], letter: 'P' })
    expect(keyOf('close', 'Linux x86_64')).toEqual({ marks: ['control', 'shift'], letter: 'W' })
  })

  it('is nothing for a command no keystroke reaches', () => {
    expect(keyOf('destroy', 'Linux x86_64')).toBeUndefined()
    expect(keysOf('destroy', 'Linux x86_64')).toEqual({})
  })
})

describe('every keystroke the table holds', () => {
  const APPLE = 'MacIntel'
  const commands = commandsOf(words, APPLE)

  it('names a command the window has', () => {
    const known = new Set(commands.map((one) => one.id))
    for (const held of CHORDS) expect(known.has(held.command)).toBe(true)
  })

  it('is drawn on the row of the command it names', () => {
    for (const held of CHORDS) {
      const command = commands.find((one) => one.id === held.command)
      expect(command?.keys).toEqual(keyOf(held.command, APPLE))
    }
  })

  it('is the only chord holding that letter with that half of the pair', () => {
    const held = CHORDS.map((one) => `${one.letter}${one.shift ? '+shift' : ''}`)
    expect(new Set(held).size).toBe(held.length)
  })

  it('leaves alone the two letters that put the palette up', () => {
    const alone = CHORDS.filter((one) => !one.shift).map((one) => one.letter)
    expect(alone).not.toContain('k')
    expect(alone).not.toContain('p')
  })

  it('reaches nothing that removes a note, a file or a vault', () => {
    const destroys = ['remove', 'destroy', 'forgetVault', 'eraseVault']
    for (const held of CHORDS) expect(destroys).not.toContain(held.command)
  })

  /**
   * What the editor and the window's own menu answer to. A chord in the table
   * that either of them takes first is a key drawn over silence.
   */
  it('takes no letter the editor or the application menu has taken', () => {
    // `Shift-Mod-k` and `Shift-Mod-\` in @codemirror/commands, and `Mod-Shift-u`
    // and `Mod-Shift-z` on Apple keyboards.
    const editor = ['k', '\\', 'u', 'z']
    // Redo and force reload, the whole of what the default menu holds with Shift.
    const menu = ['z', 'r']
    for (const held of CHORDS.filter((one) => one.shift)) {
      expect(editor).not.toContain(held.letter)
      expect(menu).not.toContain(held.letter)
    }
  })
})
