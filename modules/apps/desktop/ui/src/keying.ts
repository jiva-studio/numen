/**
 * The keystrokes that reach a command away from the palette.
 *
 * A chord stands here once: the window reads it to know what a keystroke asks
 * for, and the commands read it to write the key on the row that names one, so
 * a key a person sees drawn is a key that works. Control and Command are the
 * same key, and a chord holding Alt is nobody's.
 *
 * The two the window keeps for itself are not here. `k` puts up the search and
 * `p` the commands; neither is a command being carried out, and each is that
 * letter with Shift left alone.
 */
import { keyChord, type PaletteKeys } from '@numen/ui'

/** One command a keystroke reaches. */
export interface Chord {
  /** What it asks for, by the identity the commands give it. */
  readonly command: string
  /** The letter it is held with. */
  readonly letter: string
  /** Whether Shift is held with it. */
  readonly shift: boolean
}

/**
 * Every keystroke that carries a command out.
 *
 * A command earns one by being reached often and by leaving nothing behind that
 * a person would have to undo. Nothing that removes a note, a vault or a file is
 * ever one keystroke away. A letter means one word wherever it is written, and
 * Shift is held where the letter on its own is spoken for.
 */
export const CHORDS: readonly Chord[] = [
  { command: 'note', letter: 'n', shift: false },
  { command: 'goto', letter: 'g', shift: false },
  { command: 'travel', letter: 'p', shift: true },
  { command: 'plex', letter: 'x', shift: true },
  { command: 'child', letter: 'c', shift: true },
  { command: 'agent', letter: 'a', shift: true },
  { command: 'close', letter: 'w', shift: true },
]

/** Whether this keystroke is one the window answers at all. */
export const chorded = (event: {
  readonly altKey: boolean
  readonly ctrlKey: boolean
  readonly metaKey: boolean
}): boolean => !event.altKey && (event.ctrlKey || event.metaKey)

/** What a letter asks for, and nothing where no command answers to it. */
export const commandFor = (letter: string, shift: boolean): string =>
  CHORDS.find((one) => one.letter === letter.toLowerCase() && one.shift === shift)?.command ?? ''

/**
 * How a command's keystroke is drawn on the keyboard in hand, and nothing for
 * a command no keystroke reaches.
 */
export const keyOf = (command: string, agent: string): PaletteKeys | undefined => {
  const chord = CHORDS.find((one) => one.command === command)
  return chord ? keyChord(chord.letter, agent, chord.shift) : undefined
}

/** That keystroke as a row of the palette takes it. */
export const keysOf = (command: string, agent: string): { keys?: PaletteKeys } => {
  const drawn = keyOf(command, agent)
  return drawn ? { keys: drawn } : {}
}
