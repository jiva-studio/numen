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
