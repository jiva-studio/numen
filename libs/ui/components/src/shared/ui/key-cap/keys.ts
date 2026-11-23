/** What a cap draws, as plain values. */

import { overlayIcon } from './overlayIcon'

/** A key that is held down. A cap draws each of these as an icon. */
export type KeyIcon = 'control' | 'shift' | 'command' | 'option' | 'return'

/**
 * One keystroke as it is drawn: the keys held, in the order they are read, and
 * the letter held with them. A keystroke that is icons alone carries no letter.
 */
export interface Keystroke {
  readonly icons: readonly KeyIcon[]
  readonly letter: string
}

/**
 * A keystroke of one letter and the key beside the space bar, as it is drawn.
 */
export const createKeyChord = (letter: string, agent: string, shift = false): Keystroke => ({
  icons: shift ? [overlayIcon(agent), 'shift'] : [overlayIcon(agent)],
  letter: letter.toUpperCase(),
})
