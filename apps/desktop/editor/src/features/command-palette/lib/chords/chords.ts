/**
 * The keystrokes that reach a command away from the palette.
 *
 * Each chord is defined once: the window reads it to map a keystroke to a
 * command, and the commands read it to show the key on their palette row.
 * Control and Command are equivalent, and a chord that includes Alt is not
 * a chord.
 *
 * `k` (search) and `p` (commands) are not listed; each is that letter with
 * Shift unpressed.
 */
import { keyChord, type PaletteKeys } from '@numen/ui'

/** One command bound to a keystroke. */
export interface Chord {
  /** The command identifier. */
  readonly command: string
  /** The letter pressed with the modifier. */
  readonly letter: string
  /** Whether Shift is pressed. */
  readonly hasShift: boolean
}

export const CHORDS: readonly Chord[] = [
  { command: 'note', letter: 'n', hasShift: false },
  { command: 'newVault', letter: 'n', hasShift: true },
  { command: 'goto', letter: 'g', hasShift: false },
  { command: 'child', letter: 'c', hasShift: true },
  { command: 'close', letter: 'w', hasShift: true },
]

/** Whether this keystroke is a chord that the window handles. */
export const isChord = (event: {
  readonly altKey: boolean
  readonly ctrlKey: boolean
  readonly metaKey: boolean
}): boolean => !event.altKey && (event.ctrlKey || event.metaKey)

/** The command bound to a letter, or an empty string when none is. */
export const findChordCommand = (letter: string, hasShift: boolean): string =>
  CHORDS.find((one) => one.letter === letter.toLowerCase() && one.hasShift === hasShift)?.command ??
  ''

/**
 * The keys of a command's chord as drawn for the current platform, or undefined
 * for a command without a chord.
 */
export const findChordKeys = (command: string, agent: string): PaletteKeys | undefined => {
  const chord = CHORDS.find((one) => one.command === command)
  return chord ? keyChord(chord.letter, agent, chord.hasShift) : undefined
}

/** The chord as a palette row property. */
export const createRowKeys = (command: string, agent: string): { keys?: PaletteKeys } => {
  const chordKeys = findChordKeys(command, agent)
  return chordKeys ? { keys: chordKeys } : {}
}

/** The keys that hold the window's modifier in a shortcut a plugin declares, such as `mod+shift+x`. */
const MODIFIERS: ReadonlySet<string> = new Set(['mod', 'ctrl', 'control', 'cmd'])

/**
 * The keys of a plugin-declared shortcut as drawn for the current platform, or
 * undefined unless the shortcut is the window's modifier plus a letter,
 * optionally with Shift.
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
