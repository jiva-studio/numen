/**
 * The keystrokes that reach a command away from the palette.
 *
 * A chord stands here once: the window reads it to know what a keystroke asks
 * for, and the commands read it to write the key on the row that names one, so
 * a key a person sees drawn is a key that works. Control and Command are the
 * same key, and a chord holding Alt is nobody's.
 *
 * The two the window keeps for itself are not here. `k` puts up the search and
 * `p` the commands; neither is a command being carried out.
 */
import { keyWord } from '@numen/ui'

/** One command a keystroke reaches. */
export interface Chord {
  /** What it asks for, by the identity the commands give it. */
  readonly command: string
  /** The letter it is held with. */
  readonly letter: string
}

/**
 * Every keystroke that carries a command out.
 *
 * A command earns one by being reached often and by leaving nothing behind that
 * a person would have to undo. Nothing that removes a note, a vault or a file is
 * ever one keystroke away.
 */
export const CHORDS: readonly Chord[] = [
  { command: 'note', letter: 'n' },
  { command: 'goto', letter: 'g' },
]

/** Whether this keystroke is one the window answers at all. */
export const chorded = (event: {
  readonly altKey: boolean
  readonly ctrlKey: boolean
  readonly metaKey: boolean
}): boolean => !event.altKey && (event.ctrlKey || event.metaKey)

/** What a letter asks for, and nothing where no command answers to it. */
export const commandFor = (letter: string): string =>
  CHORDS.find((one) => one.letter === letter.toLowerCase())?.command ?? ''

/** How a command's keystroke is written on the keyboard in hand. */
export const keyOf = (command: string, agent: string): string => {
  const chord = CHORDS.find((one) => one.command === command)
  return chord ? keyWord(chord.letter, agent) : ''
}
