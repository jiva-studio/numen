/** What a cap draws, as plain values. */

/** A key that is held down. A cap draws each of these as an icon. */
export type PaletteIcon = 'control' | 'shift' | 'command' | 'option' | 'return'

/**
 * One keystroke as it is drawn: the keys held, in the order they are read, and
 * the letter held with them. A keystroke that is icons alone carries no letter.
 */
export interface PaletteKeys {
  readonly icons: readonly PaletteIcon[]
  readonly letter: string
}

/**
 * The key next to the space bar on the keyboard a browser reports it is running
 * on: Command on Apple keyboards, Control everywhere else.
 */
export const overlayIcon = (agent: string): PaletteIcon =>
  /mac|iphone|ipad|ipod/i.test(agent) ? 'command' : 'control'

/**
 * A keystroke of one letter and the key beside the space bar, as it is drawn.
 */
export const keyChord = (letter: string, agent: string, shift = false): PaletteKeys => ({
  icons: shift ? [overlayIcon(agent), 'shift'] : [overlayIcon(agent)],
  letter: letter.toUpperCase(),
})
