/**
 * The keys drawn beside a command, tested without a window. A key drawn on a row that does nothing is
 * the fault guarded against: the keybindings of the application are read to bind and to draw.
 */
import { describe, expect, it } from 'vitest'
import { APP_KEYBINDINGS } from '@numen/plugin-host'
import { createCommands } from '../commands/commands'
import { WORDS as words } from '@/shared/words'
import { createRowKeys, findChordKeys, findShortcutKeys } from './keys'

describe('how a key is drawn on the row that names it', () => {
  it('is the key the keyboard in hand holds, and the letter in capitals', () => {
    expect(findChordKeys('app:note.create', 'MacIntel')).toEqual({
      icons: ['command'],
      letter: 'N',
    })
    expect(findChordKeys('app:note.create', 'Linux x86_64')).toEqual({
      icons: ['control'],
      letter: 'N',
    })
    expect(findChordKeys('app:file.open#note', 'Linux x86_64')).toEqual({
      icons: ['control'],
      letter: 'G',
    })
  })

  it('carries Shift where the key holds it', () => {
    expect(findChordKeys('app:note.createChild', 'MacIntel')).toEqual({
      icons: ['command', 'shift'],
      letter: 'C',
    })
    expect(findChordKeys('app:tab.close', 'Linux x86_64')).toEqual({
      icons: ['control', 'shift'],
      letter: 'W',
    })
  })

  it('is the key of the command with the same values only', () => {
    expect(findChordKeys('app:file.open#note', 'Linux x86_64', { placement: 'split' })).toBeUndefined()
    expect(createRowKeys('app:file.open#note', 'Linux x86_64', { placement: 'split' })).toEqual({})
    expect(createRowKeys('app:file.open#note', 'Linux x86_64', {})).toEqual({
      keys: { icons: ['control'], letter: 'G' },
    })
  })

  it('is nothing for a command no key reaches', () => {
    expect(findChordKeys('app:file.remove#permanent', 'Linux x86_64')).toBeUndefined()
    expect(createRowKeys('app:file.remove#permanent', 'Linux x86_64')).toEqual({})
  })
})

describe('how a key a plugin declares is drawn', () => {
  it('is the modifier of the window with the letter, and Shift where the key names it', () => {
    expect(findShortcutKeys('mod+shift+x', 'MacIntel')).toEqual({
      icons: ['command', 'shift'],
      letter: 'X',
    })
    expect(findShortcutKeys('Ctrl + G', 'Linux x86_64')).toEqual({
      icons: ['control'],
      letter: 'G',
    })
  })

  it('is nothing for a key the palette draws no keystroke of', () => {
    expect(findShortcutKeys('shift+x', 'MacIntel')).toBeUndefined()
    expect(findShortcutKeys('mod+alt+x', 'MacIntel')).toBeUndefined()
    expect(findShortcutKeys('mod+f5', 'MacIntel')).toBeUndefined()
    expect(findShortcutKeys('mod+enter', 'MacIntel')).toBeUndefined()
    expect(findShortcutKeys('', 'MacIntel')).toBeUndefined()
  })
})

describe('every key the application holds', () => {
  const APPLE = 'MacIntel'
  const OPENS_A_PALETTE = ['app:vault.search', 'app:commands.open']
  const appKeys = APP_KEYBINDINGS.filter((one) => !OPENS_A_PALETTE.includes(one.command))

  it('names a row the palette has, unless it opens a palette', () => {
    const recognized = new Set(createCommands(words, APPLE).map((one) => one.id))
    for (const one of appKeys) expect(recognized.has(one.command)).toBe(true)
  })

  it('is drawn on the row of the command it names', () => {
    const commands = createCommands(words, APPLE)
    for (const one of appKeys) {
      const row = commands.find((command) => command.id === one.command)
      expect(row?.keys).toEqual(findChordKeys(one.command, APPLE))
    }
  })

  it('is the only key holding that chord under that condition', () => {
    const chords = APP_KEYBINDINGS.map((one) => `${one.key}|${one.when}`)
    expect(new Set(chords).size).toBe(chords.length)
  })

  it('reaches nothing that removes a note, a file or a vault', () => {
    const destroys = [
      'app:file.remove',
      'app:file.remove#permanent',
      'app:vault.remove',
      'app:vault.erase',
    ]
    for (const one of APP_KEYBINDINGS) expect(destroys).not.toContain(one.command)
  })

  /**
   * What the editor and the window's own menu answer to. A key that either of them takes first is a
   * key drawn over silence.
   */
  it('takes no letter the editor or the application menu has taken', () => {
    // `Shift-Mod-k` and `Shift-Mod-\` in @codemirror/commands, and `Mod-Shift-u` and `Mod-Shift-z`
    // on Apple keyboards.
    const editor = ['k', '\\', 'u', 'z']
    // Redo and force reload, the whole of what the default menu holds with Shift.
    const menu = ['z', 'r']
    for (const one of APP_KEYBINDINGS.filter((key) => key.key.includes('shift'))) {
      const letter = one.key.split('+').at(-1) ?? ''
      expect(editor).not.toContain(letter)
      expect(menu).not.toContain(letter)
    }
  })
})
