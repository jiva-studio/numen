/**
 * The keystrokes that reach a command away from the palette.
 *
 * A chord stands here once: the window reads it to know what a keystroke asks
 * for, and the commands read it to write the key on the row that names one.
 * Control and Command are the same key, and a chord holding Alt is nobody's.
 *
 * `k` sets up the search and `p` the commands; neither is here, and each is
 * that letter with Shift left alone.
 */
import { keyChord, type PaletteKeys } from '@numen/ui'

/** One command a keystroke reaches. */
export interface Chord {
  /** What it asks for, by the identity the commands give it. */
  readonly command: string
  /** The letter it is held with. */
  readonly letter: string
  /** Whether Shift is held with it. */
  readonly hasShift: boolean
}

export const CHORDS: readonly Chord[] = [
  { command: 'note', letter: 'n', hasShift: false },
  { command: 'newVault', letter: 'n', hasShift: true },
  { command: 'goto', letter: 'g', hasShift: false },
  { command: 'child', letter: 'c', hasShift: true },
  { command: 'close', letter: 'w', hasShift: true },
]

/** Whether this keystroke is one the window answers at all. */
export const isChord = (event: {
  readonly altKey: boolean
  readonly ctrlKey: boolean
  readonly metaKey: boolean
}): boolean => !event.altKey && (event.ctrlKey || event.metaKey)

/** What a letter asks for, and nothing where no command answers to it. */
export const findChordCommand = (letter: string, hasShift: boolean): string =>
  CHORDS.find((one) => one.letter === letter.toLowerCase() && one.hasShift === hasShift)?.command ??
  ''

/**
 * How a command's keystroke is drawn on the keyboard in hand, and nothing for
 * a command no keystroke reaches.
 */
export const findChordKeys = (command: string, agent: string): PaletteKeys | undefined => {
  const chord = CHORDS.find((one) => one.command === command)
  return chord ? keyChord(chord.letter, agent, chord.hasShift) : undefined
}

/** That keystroke as a row of the palette takes it. */
export const createRowKeys = (command: string, agent: string): { keys?: PaletteKeys } => {
  const chordKeys = findChordKeys(command, agent)
  return chordKeys ? { keys: chordKeys } : {}
}

/** The keys that hold the window's modifier in a shortcut a plugin declares, such as `mod+shift+x`. */
const MODIFIERS: ReadonlySet<string> = new Set(['mod', 'ctrl', 'control', 'cmd'])

/**
 * How a shortcut a plugin declares is drawn on the keyboard in hand, and nothing
 * for one that is not the window's modifier with a letter, and Shift or not.
 */
export const findShortcutKeys = (shortcut: string, agent: string): PaletteKeys | undefined => {
  const parts = shortcut
    .toLowerCase()
    .split('+')
    .map((part) => part.trim())
  const letter = parts.at(-1) ?? ''
  const modifiers = parts.slice(0, -1)
  if (!/^[a-z]$/.test(letter) || !modifiers.some((one) => MODIFIERS.has(one))) return undefined
  if (modifiers.some((one) => !MODIFIERS.has(one) && one !== 'shift')) return undefined
  return keyChord(letter, agent, modifiers.includes('shift'))
}
