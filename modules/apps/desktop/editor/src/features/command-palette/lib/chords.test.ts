/**
 * The keystrokes that reach a command, asked without a window.
 *
 * The failure this is here for is a key drawn on a row that does nothing: the
 * table is read twice, once to bind and once to draw, and a letter naming a
 * command the window does not have would draw a cap over silence.
 */
import { describe, expect, it } from 'vitest'
import { isChord, findChordCommand, findChordKeys, createRowKeys, CHORDS } from './chords'
import { createCommands } from './commands'
import { WORDS as words } from '@/shared/words'

const createKeyEvent = (over: Partial<KeyboardEventInit> = {}) => ({
  altKey: false,
  ctrlKey: false,
  metaKey: false,
  ...over,
})

describe('a keystroke the window answers', () => {
  it('is one letter retained with Control, or with Command', () => {
    expect(isChord(createKeyEvent({ ctrlKey: true }))).toBe(true)
    expect(isChord(createKeyEvent({ metaKey: true }))).toBe(true)
  })

  it('belongs to nobody while Alt is retained with it', () => {
    expect(isChord(createKeyEvent({ ctrlKey: true, altKey: true }))).toBe(false)
    expect(isChord(createKeyEvent({ metaKey: true, altKey: true }))).toBe(false)
  })

  it('belongs to nobody with neither of the two retained', () => {
    expect(isChord(createKeyEvent())).toBe(false)
  })
})

describe('the command a letter asks for', () => {
  it('is the one the chord alongside it names', () => {
    expect(findChordCommand('n', false)).toBe('note')
    expect(findChordCommand('g', false)).toBe('goto')
  })

  it('is the same command in either case', () => {
    expect(findChordCommand('N', false)).toBe('note')
  })

  it('is another command entirely when Shift is retained with the letter', () => {
    expect(findChordCommand('p', true)).toBe('travel')
    expect(findChordCommand('c', true)).toBe('child')
    expect(findChordCommand('a', true)).toBe('agent')
    expect(findChordCommand('w', true)).toBe('close')
    expect(findChordCommand('n', true)).toBe('newVault')
  })

  it('is nothing when a letter is retained with the wrong half of the chord', () => {
    expect(findChordCommand('g', true)).toBe('')
    expect(findChordCommand('c', false)).toBe('')
    expect(findChordCommand('w', false)).toBe('')
  })

  it('is nothing where no command answers to the letter', () => {
    expect(findChordCommand('q', false)).toBe('')
    expect(findChordCommand('q', true)).toBe('')
  })

  it('is nothing for the two letters the window keeps for itself', () => {
    expect(findChordCommand('k', false)).toBe('')
    expect(findChordCommand('p', false)).toBe('')
  })
})

describe('how a keystroke is drawn on the row that names it', () => {
  it('is the key the keyboard in hand holds, and the letter in capitals', () => {
    expect(findChordKeys('note', 'MacIntel')).toEqual({ icons: ['command'], letter: 'N' })
    expect(findChordKeys('note', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'N' })
    expect(findChordKeys('goto', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('carries Shift where the chord holds it', () => {
    expect(findChordKeys('travel', 'MacIntel')).toEqual({
      icons: ['command', 'shift'],
      letter: 'P',
    })
    expect(findChordKeys('close', 'Linux x86_64')).toEqual({
      icons: ['control', 'shift'],
      letter: 'W',
    })
  })

  it('is nothing for a command no keystroke reaches', () => {
    expect(findChordKeys('destroy', 'Linux x86_64')).toBeUndefined()
    expect(createRowKeys('destroy', 'Linux x86_64')).toEqual({})
  })
})

describe('every keystroke the table holds', () => {
  const APPLE = 'MacIntel'
  const createAppleCommands = () => createCommands(words, APPLE)

  it('names a command the window has', () => {
    const commands = createAppleCommands()
    const recognized = new Set(commands.map((one) => one.id))
    for (const target of CHORDS) expect(recognized.has(target.command)).toBe(true)
  })

  it('is drawn on the row of the command it names', () => {
    const commands = createAppleCommands()
    for (const target of CHORDS) {
      const command = commands.find((one) => one.id === target.command)
      expect(command?.keys).toEqual(findChordKeys(target.command, APPLE))
    }
  })

  it('is the only chord holding that letter with that half of the pair', () => {
    const target = CHORDS.map((one) => `${one.letter}${one.hasShift ? '+shift' : ''}`)
    expect(new Set(target).size).toBe(target.length)
  })

  it('leaves alone the two letters that put the palette up', () => {
    const alone = CHORDS.filter((one) => !one.hasShift).map((one) => one.letter)
    expect(alone).not.toContain('k')
    expect(alone).not.toContain('p')
  })

  it('reaches nothing that removes a note, a file or a vault', () => {
    const destroys = ['remove', 'destroy', 'removeVault', 'eraseVault']
    for (const target of CHORDS) expect(destroys).not.toContain(target.command)
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
    for (const target of CHORDS.filter((one) => one.hasShift)) {
      expect(editor).not.toContain(target.letter)
      expect(menu).not.toContain(target.letter)
    }
  })
})
